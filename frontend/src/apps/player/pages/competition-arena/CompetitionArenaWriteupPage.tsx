import {
  CloudUploadOutlined,
  DownloadOutlined,
  FileDoneOutlined,
  FileTextOutlined,
  LoginOutlined,
} from "@ant-design/icons";
import { App, Button, Spin, Typography } from "antd";
import { useEffect, useRef, useState, type ChangeEvent } from "react";
import { getErrorMessage } from "@shared/api/http";
import { playerApi, type PlayerWriteup } from "@shared/api/player";
import { useCompetitionArena } from "../PlayerCompetitionArenaPage";
import { formatDate } from "./TheoryCompetitionExamPanel";

function isWriteup(value: PlayerWriteup | { data: null; message?: string }): value is PlayerWriteup {
  return Boolean((value as PlayerWriteup)?.id);
}

function formatFileSize(size?: number) {
  const value = Number(size ?? 0);
  if (!Number.isFinite(value) || value <= 0) return "未知大小";
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

export function CompetitionArenaWriteupPage() {
  const { message } = App.useApp();
  const {
    competitionID,
    competition,
    canEnter,
    loggedIn,
    registrationState,
  } = useCompetitionArena();
  const inputRef = useRef<HTMLInputElement | null>(null);
  const requestedRef = useRef<number | null>(null);
  const [writeup, setWriteup] = useState<PlayerWriteup | null>(null);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);

  async function loadWriteup() {
    if (!competitionID || !loggedIn || !canEnter) return;
    setLoading(true);
    try {
      const result = await playerApi.competitionWriteup(competitionID);
      setWriteup(isWriteup(result) ? result : null);
    } catch (error) {
      message.error(getErrorMessage(error, "获取 Writeup 失败"));
      setWriteup(null);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!competitionID || !loggedIn || !canEnter) return;
    if (requestedRef.current === competitionID) return;
    requestedRef.current = competitionID;
    void loadWriteup();
  }, [competitionID, loggedIn, canEnter]);

  function handleUploadClick() {
    if (!loggedIn) {
      window.dispatchEvent(
        new CustomEvent("show-player-auth", { detail: "login" }),
      );
      message.info("请先登录后再上传 Writeup");
      return;
    }
    if (!canEnter) {
      message.warning(registrationState.description || "报名后才能上传 Writeup");
      return;
    }
    inputRef.current?.click();
  }

  async function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file || !competitionID) return;

    setUploading(true);
    try {
      const formData = new FormData();
      formData.append("file", file);
      formData.append("title", file.name);
      const result = await playerApi.uploadCompetitionWriteup(
        competitionID,
        formData,
      );
      setWriteup(result);
      requestedRef.current = competitionID;
      message.success("Writeup 上传成功");
    } catch (error) {
      message.error(getErrorMessage(error, "Writeup 上传失败"));
    } finally {
      setUploading(false);
    }
  }

  return (
    <section className="arena-writeup-page">
      <input
        ref={inputRef}
        type="file"
        accept=".md,.markdown,.pdf,.doc,.docx,.zip,.7z,.rar,.txt"
        className="arena-writeup-file-input"
        onChange={(event) => void handleFileChange(event)}
      />

      <div className="arena-writeup-hero">
        <div>
          <div className="arena-writeup-kicker">Competition Writeup</div>
          <Typography.Title level={2}>Writeup 提交</Typography.Title>
          <Typography.Paragraph>
            {competition?.name
              ? `${competition.name} 的 Writeup 文件会保存在当前比赛记录中。`
              : "上传并管理当前比赛的 Writeup 文件。"}
          </Typography.Paragraph>
        </div>
        <Button
          type="primary"
          icon={<CloudUploadOutlined />}
          loading={uploading}
          onClick={handleUploadClick}
        >
          {writeup ? "重新上传" : "上传 Writeup"}
        </Button>
      </div>

      <Spin spinning={loading}>
        {writeup ? (
          <div className="arena-writeup-card">
            <div className="arena-writeup-file-icon">
              <FileDoneOutlined />
            </div>
            <div className="arena-writeup-file-copy">
              <span>当前提交</span>
              <strong>{writeup.title || writeup.file_name}</strong>
              <p>
                {writeup.file_name} · {formatFileSize(writeup.file_size ?? writeup.size)} ·{" "}
                {formatDate(writeup.updated_at || writeup.created_at)}
              </p>
            </div>
            {writeup.url ? (
              <Button
                icon={<DownloadOutlined />}
                href={writeup.url}
                target="_blank"
              >
                下载查看
              </Button>
            ) : null}
          </div>
        ) : (
          <div className="arena-writeup-empty">
            <div className="arena-writeup-empty-icon">
              {loggedIn ? <FileTextOutlined /> : <LoginOutlined />}
            </div>
            <strong>{loggedIn ? "还没有提交 Writeup" : "登录后提交 Writeup"}</strong>
            <span>
              {canEnter
                ? "上传后你可以在这里看到文件、更新时间和下载入口。"
                : registrationState.description}
            </span>
            <Button
              type="primary"
              icon={<CloudUploadOutlined />}
              loading={uploading}
              onClick={handleUploadClick}
            >
              {loggedIn ? "选择文件" : "登录后上传"}
            </Button>
          </div>
        )}
      </Spin>
    </section>
  );
}
