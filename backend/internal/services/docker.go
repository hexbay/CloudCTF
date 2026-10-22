package services

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	imagetypes "github.com/docker/docker/api/types/image"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var dockerImageReferencePattern = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*(\/[a-z0-9]+([._-][a-z0-9]+)*)*([:@][A-Za-z0-9_][A-Za-z0-9_.-]{0,127})?$`)

func DockerClientForEndpoint(endpoint string) (*client.Client, error) {
	opts := []client.Opt{
		client.WithHost(endpoint),
		client.WithAPIVersionNegotiation(),
	}
	return client.NewClientWithOpts(opts...)
}

func dockerExposedPorts(portSet map[string]struct{}) []string {
	if len(portSet) == 0 {
		return nil
	}
	ports := make([]string, 0, len(portSet))
	for port := range portSet {
		parts := strings.SplitN(string(port), "/", 2)
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			ports = append(ports, strings.TrimSpace(parts[0]))
		}
	}
	sort.Strings(ports)
	return uniqueStrings(ports)
}
func BuildDockerfile(ctx context.Context, cli *client.Client, name string, tar io.Reader, dockerfilePath string) error {
	return BuildDockerfileWithLog(ctx, cli, name, tar, dockerfilePath, nil)
}

func BuildDockerfileWithLog(ctx context.Context, cli *client.Client, name string, tar io.Reader, dockerfilePath string, onLine func(string)) error {
	var tags []string
	if name != "" {
		tags = []string{name}
	}
	dockerfilePath = filepath.ToSlash(strings.TrimSpace(dockerfilePath))
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}
	resp, err := cli.ImageBuild(ctx, tar, build.ImageBuildOptions{
		Tags:       tags,
		Remove:     true,
		Dockerfile: dockerfilePath,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	for {
		var line map[string]any
		if err := decoder.Decode(&line); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}
		if stream, ok := line["stream"].(string); ok && stream != "" {
			emitDockerLogLines(stream, onLine)
		}
		if errMsg, ok := line["error"].(string); ok && errMsg != "" {
			emitDockerLogLines(errMsg, onLine)
			return errors.New(errMsg)
		}
	}
	return nil

}

func BuildDockerfileByPath(ctx context.Context, cli *client.Client, name string, dockerDirectory string, dockerfilePath string) error {
	return BuildDockerfileByPathWithLog(ctx, cli, name, dockerDirectory, dockerfilePath, nil)
}

func BuildDockerfileByPathWithLog(ctx context.Context, cli *client.Client, name string, dockerDirectory string, dockerfilePath string, onLine func(string)) error {
	tarReader, err := CreateDockerTar(dockerDirectory)
	if err != nil {
		return err
	}
	return BuildDockerfileWithLog(ctx, cli, name, tarReader, dockerfilePath, onLine)
}

func emitDockerLogLines(output string, onLine func(string)) {
	if onLine == nil {
		return
	}
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			onLine(line)
		}
	}
}

func StreamDockerImagePull(ctx context.Context, endpoint, imageName string, onLine func(string)) error {
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return err
	}
	defer cli.Close()

	reader, err := cli.ImagePull(ctx, imageName, imagetypes.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		onLine(scanner.Text())
	}
	return scanner.Err()
}

func PullDockerImage(ctx context.Context, endpoint, imageName string) ([]string, error) {
	logs := make([]string, 0, 32)
	err := StreamDockerImagePull(ctx, endpoint, imageName, func(line string) {
		logs = append(logs, line)
	})
	return logs, err
}

func InspectDockerImageExposedPorts(ctx context.Context, endpoint, imageName string) ([]string, error) {
	if strings.TrimSpace(imageName) == "" {
		return nil, nil
	}
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	imageInfo, err := cli.ImageInspect(ctx, imageName)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such image") {
			if _, pullErr := PullDockerImage(ctx, endpoint, imageName); pullErr != nil {
				return nil, pullErr
			}
			imageInfo, err = cli.ImageInspect(ctx, imageName)
		}
		if err != nil {
			return nil, err
		}
	}
	if imageInfo.Config == nil || len(imageInfo.Config.ExposedPorts) == 0 {
		return nil, nil
	}
	return dockerExposedPorts(imageInfo.Config.ExposedPorts), nil
}

func BuildDockerImage(ctx context.Context, endpoint, dockerfile, tag string, buildArgs map[string]*string) ([]string, error) {
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	if err := tw.WriteHeader(&tar.Header{
		Name: "Dockerfile",
		Mode: 0o600,
		Size: int64(len(dockerfile)),
	}); err != nil {
		return nil, err
	}
	if _, err := tw.Write([]byte(dockerfile)); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}

	resp, err := cli.ImageBuild(ctx, bytes.NewReader(buf.Bytes()), build.ImageBuildOptions{
		Tags:       []string{tag},
		Remove:     true,
		BuildArgs:  buildArgs,
		Dockerfile: "Dockerfile",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	logs := make([]string, 0, 32)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		logs = append(logs, scanner.Text())
	}
	return logs, scanner.Err()
}

func LoadDockerImage(ctx context.Context, endpoint string, reader io.Reader, tag string) ([]string, error) {
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	resp, err := cli.ImageLoad(ctx, reader, client.ImageLoadWithQuiet(true))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	logs := []string{string(body)}
	if tag != "" {
		logs = append(logs, "requested tag: "+tag)
	}
	return logs, nil
}

func CreateDockerTar(dockerDirectory string) (io.Reader, error) {
	dockerDirectory = strings.TrimSpace(dockerDirectory)
	if dockerDirectory == "" {
		return nil, fmt.Errorf("docker directory is required")
	}
	info, err := os.Stat(dockerDirectory)
	if err != nil {
		return nil, fmt.Errorf("stat docker directory failed: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("docker directory is not a directory: %s", dockerDirectory)
	}
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	walkErr := filepath.Walk(dockerDirectory, func(path string, fileInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == dockerDirectory {
			return nil
		}

		relPath, err := filepath.Rel(dockerDirectory, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		header, err := tar.FileInfoHeader(fileInfo, "")
		if err != nil {
			return fmt.Errorf("create tar header failed for %s: %w", relPath, err)
		}
		header.Name = relPath
		if fileInfo.IsDir() {
			if !strings.HasSuffix(header.Name, "/") {
				header.Name += "/"
			}
			if err := tw.WriteHeader(header); err != nil {
				return fmt.Errorf("write tar dir header failed for %s: %w", relPath, err)
			}
			return nil
		}
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("write tar file header failed for %s: %w", relPath, err)
		}
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open file failed for %s: %w", relPath, err)
		}
		defer file.Close()
		if _, err := io.Copy(tw, file); err != nil {
			return fmt.Errorf("write tar file content failed for %s: %w", relPath, err)
		}
		return nil
	})
	if walkErr != nil {
		_ = tw.Close()
		return nil, walkErr
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("close tar writer failed: %w", err)
	}
	return buf, nil
}

func MapToDockerFilters(key string, values []string) filters.Args {
	args := filters.NewArgs()
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		args.Add(key, value)
	}
	return args
}

func NormalizeContainerPort(value string) string {
	text := strings.TrimSpace(strings.Trim(value, `"'`))
	if text == "" {
		return ""
	}
	if strings.Contains(text, "/") {
		return text
	}
	return text + "/tcp"
}

func PortNumber(value string) string {
	text := NormalizeContainerPort(value)
	parts := strings.SplitN(text, "/", 2)
	return parts[0]
}

func PortProtocol(value string) string {
	text := NormalizeContainerPort(value)
	parts := strings.SplitN(text, "/", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		return parts[1]
	}
	return "tcp"
}

func EnsureDockerNetwork(ctx context.Context, endpoint, networkName string, labels map[string]string) error {
	if strings.TrimSpace(networkName) == "" {
		return nil
	}
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return err
	}
	defer cli.Close()
	if _, err := cli.NetworkInspect(ctx, networkName, networktypes.InspectOptions{}); err == nil {
		return nil
	}
	_, err = cli.NetworkCreate(ctx, networkName, networktypes.CreateOptions{
		Driver: "bridge",
		Labels: labels,
	})
	return err
}

type DockerContainerCreateOptions struct {
	Name        string
	Image       string
	Ports       map[string]string
	Environment map[string]string
	Command     []string
	NetworkName string
	Labels      map[string]string
}

func ListDockerContainers(ctx context.Context, endpoint string, hostID uint, hostName, hostOutIP string, dockerFilters filters.Args) ([]map[string]any, error) {
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	items, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: dockerFilters,
	})
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		name := ""
		if len(item.Names) > 0 {
			name = strings.TrimPrefix(item.Names[0], "/")
		}
		ports := map[string]any{}
		for _, port := range item.Ports {
			hostPort := strconv.Itoa(int(port.PublicPort))
			if hostPort == "0" {
				hostPort = strconv.Itoa(int(port.PrivatePort))
			}
			ports[hostPort] = fmt.Sprintf("%d/%s", port.PrivatePort, port.Type)
		}
		rows = append(rows, map[string]any{
			"host_id":      hostID,
			"host_name":    hostName,
			"out_ip":       hostOutIP,
			"id":           item.ID,
			"container_id": item.ID,
			"name":         name,
			"image":        item.Image,
			"status":       item.State,
			"state":        item.State,
			"created":      fmt.Sprint(item.Created),
			"created_at":   item.Created,
			"ports":        ports,
			"command":      item.Command,
			"labels":       item.Labels,
		})
	}
	return rows, nil
}

func CreateDockerContainerWithOptions(
	ctx context.Context,
	endpoint string,
	options DockerContainerCreateOptions,
	pullMissing func(context.Context, string, string) ([]string, error),
) (map[string]any, error) {
	options.Image = strings.TrimSpace(options.Image)
	if err := ValidateDockerImageReference(options.Image); err != nil {
		return nil, err
	}
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	cfg := &container.Config{
		Image:  options.Image,
		Env:    make([]string, 0, len(options.Environment)),
		Cmd:    options.Command,
		Labels: options.Labels,
	}
	for key, value := range options.Environment {
		cfg.Env = append(cfg.Env, key+"="+value)
	}
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	for hostPort, containerPort := range options.Ports {
		portKey := NormalizeContainerPort(containerPort)
		if portKey == "" {
			continue
		}
		natPort, err := nat.NewPort(PortProtocol(portKey), PortNumber(portKey))
		if err != nil {
			return nil, err
		}
		exposedPorts[natPort] = struct{}{}
		binding := nat.PortBinding{}
		hostPort = strings.TrimSpace(hostPort)
		if hostPort != "" && hostPort != PortNumber(portKey) {
			binding.HostPort = hostPort
		}
		portBindings[natPort] = append(portBindings[natPort], binding)
	}
	if len(exposedPorts) > 0 {
		cfg.ExposedPorts = exposedPorts
	}
	hostCfg := &container.HostConfig{}
	if len(portBindings) > 0 {
		hostCfg.PortBindings = portBindings
	}
	if options.NetworkName != "" {
		hostCfg.NetworkMode = container.NetworkMode(options.NetworkName)
	}
	resp, err := cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, options.Name)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such image") && options.Image != "" && pullMissing != nil {
			if _, pullErr := pullMissing(ctx, endpoint, options.Image); pullErr != nil {
				return nil, pullErr
			}
			resp, err = cli.ContainerCreate(ctx, cfg, hostCfg, nil, nil, options.Name)
		}
		if err != nil {
			return nil, err
		}
	}
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		_ = cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return nil, err
	}
	inspect, err := cli.ContainerInspect(ctx, resp.ID)
	if err != nil {
		return nil, err
	}
	ports := map[string]any{}
	for privatePort, bindings := range inspect.NetworkSettings.Ports {
		if len(bindings) == 0 {
			ports[privatePort.Port()] = privatePort.Port() + "/" + privatePort.Proto()
			continue
		}
		for _, binding := range bindings {
			if strings.TrimSpace(binding.HostPort) == "" {
				ports[privatePort.Port()] = privatePort.Port() + "/" + privatePort.Proto()
				continue
			}
			ports[binding.HostPort] = privatePort.Port() + "/" + privatePort.Proto()
		}
	}
	return map[string]any{
		"id":           inspect.ID,
		"container_id": inspect.ID,
		"name":         strings.TrimPrefix(inspect.Name, "/"),
		"image":        inspect.Config.Image,
		"status":       inspect.State.Status,
		"state":        inspect.State.Status,
		"created":      inspect.Created,
		"created_at":   inspect.Created,
		"ports":        ports,
		"command":      strings.Join(inspect.Config.Cmd, " "),
		"labels":       inspect.Config.Labels,
	}, nil
}

func ValidateDockerImageReference(image string) error {
	image = strings.TrimSpace(image)
	if image == "" {
		return fmt.Errorf("docker image is empty")
	}
	if strings.ContainsAny(image, " \t\r\n") {
		return fmt.Errorf("invalid docker image reference %q: contains whitespace", image)
	}
	if !dockerImageReferencePattern.MatchString(image) {
		return fmt.Errorf("invalid docker image reference %q: use lowercase repository/name format such as nginx:latest or repo/app:tag", image)
	}
	return nil
}

func InspectDockerContainer(ctx context.Context, endpoint, containerID string, hostID uint, hostName, hostOutIP string) (map[string]any, error) {
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, err
	}
	ports := map[string]any{}
	for privatePort, bindings := range inspect.NetworkSettings.Ports {
		if len(bindings) == 0 {
			ports[privatePort.Port()] = privatePort.Port() + "/" + privatePort.Proto()
			continue
		}
		for _, binding := range bindings {
			ports[binding.HostPort] = privatePort.Port() + "/" + privatePort.Proto()
		}
	}
	return map[string]any{
		"host_id":      hostID,
		"host_name":    hostName,
		"out_ip":       hostOutIP,
		"id":           inspect.ID,
		"container_id": inspect.ID,
		"name":         strings.TrimPrefix(inspect.Name, "/"),
		"image":        inspect.Config.Image,
		"status":       inspect.State.Status,
		"state":        inspect.State.Status,
		"created":      inspect.Created,
		"created_at":   inspect.Created,
		"ports":        ports,
		"command":      strings.Join(inspect.Config.Cmd, " "),
		"labels":       inspect.Config.Labels,
	}, nil
}

func RunDockerContainerAction(ctx context.Context, endpoint, containerID, action string, managedCheck func(map[string]string) bool) error {
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return err
	}
	defer cli.Close()
	//inspect, err := cli.ContainerInspect(ctx, containerID)
	//if err != nil {
	//	return err
	//}
	//if managedCheck != nil && !managedCheck(inspect.Config.Labels) {
	//	return fmt.Errorf("container is not managed by this project")
	//}
	switch strings.ToLower(action) {
	case "start":
		return cli.ContainerStart(ctx, containerID, container.StartOptions{})
	case "stop":
		return cli.ContainerStop(ctx, containerID, container.StopOptions{})
	case "restart":
		return cli.ContainerRestart(ctx, containerID, container.StopOptions{})
	case "pause":
		return cli.ContainerPause(ctx, containerID)
	case "unpause":
		return cli.ContainerUnpause(ctx, containerID)
	case "delete", "remove":
		return cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
	default:
		return fmt.Errorf("unsupported action: %s", action)
	}
}

func RemoveDockerContainer(ctx context.Context, endpoint, containerID string, managedCheck func(map[string]string) bool) error {
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return err
	}
	defer cli.Close()
	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return err
	}
	if managedCheck != nil && !managedCheck(inspect.Config.Labels) {
		return fmt.Errorf("container is not managed by this project")
	}
	return cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}

func ExecDockerContainer(ctx context.Context, cli *client.Client, containerID string, cmd []string) error {
	resp, err := cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	})
	if err != nil {
		return err
	}
	attachResp, err := cli.ContainerExecAttach(ctx, resp.ID, container.ExecAttachOptions{})
	if err != nil {
		return err
	}
	defer attachResp.Close()
	if _, err := io.Copy(io.Discard, attachResp.Reader); err != nil {
		return err
	}
	inspectResp, err := cli.ContainerExecInspect(ctx, resp.ID)
	if err != nil {
		return err
	}
	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("container exec failed with exit code %d", inspectResp.ExitCode)
	}
	return nil
}

func ReadDockerContainerLogs(ctx context.Context, endpoint, containerID string, lines int) (string, error) {
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return "", err
	}
	defer cli.Close()
	reader, err := cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(lines),
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()
	body, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func DockerContainerStats(ctx context.Context, endpoint, containerID string) (map[string]any, error) {
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	resp, err := cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func DockerContainerStatus(ctx context.Context, endpoint, containerID string) (string, error) {
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return "", err
	}
	defer cli.Close()
	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}
	return inspect.State.Status, nil
}

func UpdateDockerContainer(ctx context.Context, endpoint, containerID string, payload map[string]any) error {
	cli, err := DockerClientForEndpoint(endpoint)
	if err != nil {
		return err
	}
	defer cli.Close()

	renameTo := cleanServiceString(payload["name"])
	if renameTo == "" {
		renameTo = cleanServiceString(payload["container_name"])
	}
	if renameTo != "" {
		if err := cli.ContainerRename(ctx, containerID, renameTo); err != nil {
			return err
		}
	}

	updateConfig := container.UpdateConfig{}
	hasUpdate := false
	if value := int64(intServiceValue(payload["cpu_shares"], 0)); value > 0 {
		updateConfig.Resources.CPUShares = value
		hasUpdate = true
	}
	if value := int64(intServiceValue(payload["memory"], 0)); value > 0 {
		updateConfig.Resources.Memory = value
		hasUpdate = true
	}
	if restartName := cleanServiceString(payload["restart_policy"]); restartName != "" {
		updateConfig.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyMode(restartName)}
		hasUpdate = true
	}
	if hasUpdate {
		if _, err := cli.ContainerUpdate(ctx, containerID, updateConfig); err != nil {
			return err
		}
	}
	return nil
}

func cleanServiceString(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func intServiceValue(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed
		}
	}
	return fallback
}
