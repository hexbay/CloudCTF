package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"cloudctf/backend/internal/bootstrap"
	"cloudctf/backend/internal/config"
	"cloudctf/backend/internal/models"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"gorm.io/gorm"
)

func runResetAdminPassword(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cfg := config.Load()
	username := cfg.FirstSuperuserUsername
	databaseURI := cfg.DatabaseURI
	passwordStdin := false
	activate := true

	flags := flag.NewFlagSet("reset-admin-password", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&username, "username", username, "administrator username")
	flags.StringVar(&databaseURI, "database-uri", databaseURI, "database URI; defaults to DATABASE_URI")
	flags.BoolVar(&passwordStdin, "password-stdin", false, "read one password line from stdin instead of prompting")
	flags.BoolVar(&activate, "activate", true, "reactivate the administrator account")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: cloudctf reset-admin-password [options]")
		fmt.Fprintln(stderr, "Resets an existing administrator password without starting the HTTP server.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("username must not be empty")
	}

	password, err := readResetPassword(stdin, stderr, passwordStdin)
	if err != nil {
		return err
	}
	if len(password) < 6 {
		return errors.New("new password must be at least 6 characters")
	}

	db, err := bootstrap.OpenDB(databaseURI)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}
	if !db.Migrator().HasTable(&models.User{}) {
		return errors.New("database does not contain the CloudCTF user table; check DATABASE_URI or --database-uri")
	}
	if err := resetAdminPassword(db, username, password, activate); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Administrator %q password reset successfully. Existing refresh sessions were revoked.\n", username)
	fmt.Fprintln(stdout, "The administrator must change the password after the next login.")
	return nil
}

func readResetPassword(stdin io.Reader, stderr io.Writer, passwordStdin bool) (string, error) {
	if passwordStdin {
		line, err := bufio.NewReader(io.LimitReader(stdin, 4097)).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read password from stdin: %w", err)
		}
		return strings.TrimRight(line, "\r\n"), nil
	}

	input, ok := stdin.(*os.File)
	if !ok || !term.IsTerminal(int(input.Fd())) {
		return "", errors.New("stdin is not a terminal; use --password-stdin")
	}
	first, err := readHiddenPassword(input, stderr, "New administrator password: ")
	if err != nil {
		return "", err
	}
	second, err := readHiddenPassword(input, stderr, "Confirm new password: ")
	if err != nil {
		return "", err
	}
	if first != second {
		return "", errors.New("passwords do not match")
	}
	return first, nil
}

func readHiddenPassword(input *os.File, output io.Writer, prompt string) (string, error) {
	fmt.Fprint(output, prompt)
	password, err := term.ReadPassword(int(input.Fd()))
	fmt.Fprintln(output)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(password), nil
}

func resetAdminPassword(db *gorm.DB, username, password string, activate bool) error {
	var user models.User
	if err := db.Preload("Role").Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("administrator %q was not found", username)
		}
		return fmt.Errorf("find administrator: %w", err)
	}
	if !user.IsSuperuser && strings.EqualFold(user.Role.Name, "player") {
		return fmt.Errorf("user %q is not an administrator", username)
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	now := time.Now()
	return db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"hashed_password":      string(hashed),
			"must_change_password": true,
		}
		if activate {
			updates["is_active"] = true
		}
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("update administrator: %w", err)
		}
		if tx.Migrator().HasTable(&models.RefreshToken{}) {
			if err := tx.Model(&models.RefreshToken{}).
				Where("user_id = ? AND revoked_at IS NULL", user.ID).
				Update("revoked_at", now).Error; err != nil {
				return fmt.Errorf("revoke refresh sessions: %w", err)
			}
		}
		return nil
	})
}
