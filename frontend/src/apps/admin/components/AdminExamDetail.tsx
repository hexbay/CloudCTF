import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  DeleteOutlined,
  EditOutlined,
  ExclamationCircleOutlined,
  ImportOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import {
  Button,
  Card,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Radio,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
  App,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { adminApi } from "@shared/api/admin";
import { getErrorMessage } from "@shared/api/http";

type QuestionType = "text" | "judge" | "single" | "multiple";
type OptionItem = { text: string; checked?: boolean };
type ExamQuestion = {
  id?: number;
  question: string;
  question_type: QuestionType;
  points: number;
  context?: Record<string, any>;
};
type TheoryQuestion = ExamQuestion & { id: number };
type ExamRecord = {
  id: number;
  competition_id?: number | null;
  title: string;
  total_score: number;
  is_active: boolean;
  context?: { questions?: ExamQuestion[]; question_ids?: number[]; time_limit?: number };
};

const typeColors: Record<QuestionType, string> = {
  text: "blue",
  judge: "green",
  single: "orange",
  multiple: "purple",
};

const typeNames: Record<QuestionType, string> = {
  text: "问答题",
  judge: "判断题",
  single: "单选题",
  multiple: "多选题",
};

const questionTypeOrder: QuestionType[] = ["single", "multiple", "judge", "text"];

export function AdminExamDetail({
  examId,
  competitionId,
  backTo,
}: {
  examId: number;
  competitionId?: number;
  backTo: string;
}) {
  const { message, modal } = App.useApp();
  const navigate = useNavigate();
  const [bankQuestions, setBankQuestions] = useState<TheoryQuestion[]>([]);
  const [exam, setExam] = useState<ExamRecord | null>(null);
  const [editingExam, setEditingExam] = useState<ExamRecord | null>(null);
  const [loading, setLoading] = useState(false);
  const [importVisible, setImportVisible] = useState(false);
  const [searchText, setSearchText] = useState("");
  const [filterType, setFilterType] = useState<string>("all");
  const [editMode, setEditMode] = useState(false);
  const [examForm] = Form.useForm();

  useEffect(() => {
    void load();
  }, [examId]);

  const currentExam = editMode ? editingExam : exam;
  const examQuestions = orderExamQuestions(currentExam?.context?.questions ?? []);

  const filteredBankQuestions = useMemo(() => {
    let rows = bankQuestions;
    if (filterType !== "all") {
      rows = rows.filter((q) => q.question_type === filterType);
    }
    if (searchText) {
      const text = searchText.toLowerCase();
      rows = rows.filter((q) => q.question.toLowerCase().includes(text));
    }
    return rows;
  }, [bankQuestions, filterType, searchText]);

  async function load() {
    setLoading(true);
    try {
      const examRequest = competitionId
        ? adminApi.get<ExamRecord | { exam?: ExamRecord | null }>(`/api-admin/exam/competitions/${competitionId}/exam`)
        : adminApi.list<ExamRecord>("/api-admin/exam", { current: 1, pageSize: 500 });
      const [statsResult, examsResult] = await Promise.all([
        adminApi.list<{ questions: TheoryQuestion[] }>("/api-admin/theory/question/stats"),
        examRequest,
      ]);
      const questions = (statsResult as unknown as { questions?: TheoryQuestion[] }).questions ?? [];
      const targetExam = competitionId
        ? normalizeCompetitionExam(examsResult)
        : normalizeRows<ExamRecord>(examsResult).find((item) => item.id === examId) ?? null;
      setBankQuestions(questions);
      setExam(targetExam ? normalizeExamRecord(targetExam, questions) : null);
      setEditingExam(null);
      setEditMode(false);
    } catch (error) {
      message.error(getErrorMessage(error, "加载试卷详情失败"));
    } finally {
      setLoading(false);
    }
  }

  function enterEditMode() {
    if (!exam) return;
    const next = deepCloneExam(exam);
    setEditingExam(next);
    examForm.setFieldsValue({
      title: next.title,
      is_active: next.is_active,
      time_limit: next.context?.time_limit ?? 3600,
    });
    setEditMode(true);
  }

  async function saveEdit() {
    if (!editingExam) return;
    try {
      const values = await examForm.validateFields();
      const questions = editingExam.context?.questions ?? [];
      await adminApi.update(`/api-admin/exam/exams/${editingExam.id}`, {
        title: values.title,
        is_active: values.is_active,
        total_score: sumQuestionPoints(questions),
        questions,
        time_limit: Number(values.time_limit ?? 3600),
      });
      message.success("试卷更新成功");
      await load();
    } catch (error) {
      message.error(getErrorMessage(error, "更新试卷失败"));
    }
  }

  function cancelEdit() {
    setEditMode(false);
    setEditingExam(null);
  }

  function handleDelete() {
    if (!exam) return;
    modal.confirm({
      title: "确认删除",
      icon: <ExclamationCircleOutlined />,
      content: "确定要删除这份试卷吗？删除后无法恢复。",
      okType: "danger",
      onOk: async () => {
        try {
          await adminApi.remove(`/api-admin/exam/${exam.id}`);
          message.success("试卷删除成功");
          navigate(backTo);
        } catch (error) {
          message.error(getErrorMessage(error, "删除试卷失败"));
        }
      },
    });
  }

  function openCreateQuestion() {
    if (!editingExam) return;
    const questions = orderExamQuestions([
      ...(editingExam.context?.questions ?? []),
      createEmptyExamQuestion(),
    ]);
    setEditingExam({
      ...editingExam,
      context: {
        ...editingExam.context,
        questions,
      },
      total_score: sumQuestionPoints(questions),
    });
  }

  function removeQuestion(index: number) {
    if (!editingExam) return;
    const questions = [...(editingExam.context?.questions ?? [])];
    questions.splice(index, 1);
    setEditingExam({
      ...editingExam,
      context: {
        ...editingExam.context,
        questions,
      },
      total_score: sumQuestionPoints(questions),
    });
  }

  function moveQuestion(index: number, direction: -1 | 1) {
    if (!editingExam) return;
    const questions = [...(editingExam.context?.questions ?? [])];
    const nextIndex = index + direction;
    if (nextIndex < 0 || nextIndex >= questions.length) return;
    const [item] = questions.splice(index, 1);
    questions.splice(nextIndex, 0, item);
    setEditingExam({
      ...editingExam,
      context: {
        ...editingExam.context,
        questions,
      },
    });
  }

  function updateQuestion(index: number, patch: Partial<ExamQuestion>) {
    if (!editingExam) return;
    const questions = [...(editingExam.context?.questions ?? [])];
    const current = questions[index];
    if (!current) return;
    questions[index] = {
      ...current,
      ...patch,
      context: patch.context ?? current.context ?? {},
    };
    const orderedQuestions =
      patch.question_type && patch.question_type !== current.question_type
        ? orderExamQuestions(questions)
        : questions;
    setEditingExam({
      ...editingExam,
      context: {
        ...editingExam.context,
        questions: orderedQuestions,
      },
      total_score: sumQuestionPoints(orderedQuestions),
    });
  }

  function updateQuestionType(index: number, type: QuestionType) {
    const base = createQuestionContextByType(type);
    updateQuestion(index, { question_type: type, context: base });
  }

  function updateOption(index: number, optionIndex: number, label: string) {
    const question = editingExam?.context?.questions?.[index];
    if (!question) return;
    const options = normalizeOptions(question).map((option, currentIndex) =>
      currentIndex === optionIndex ? { ...option, text: label } : option,
    );
    syncQuestionOptions(index, question, options);
  }

  function toggleOptionCorrect(index: number, optionIndex: number, checked: boolean) {
    const question = editingExam?.context?.questions?.[index];
    if (!question) return;
    let options = normalizeOptions(question);
    if (question.question_type === "single") {
      options = options.map((option, currentIndex) => ({
        ...option,
        checked: currentIndex === optionIndex ? checked : false,
      }));
    } else {
      options = options.map((option, currentIndex) =>
        currentIndex === optionIndex ? { ...option, checked } : option,
      );
    }
    syncQuestionOptions(index, question, options);
  }

  function addOption(index: number) {
    const question = editingExam?.context?.questions?.[index];
    if (!question) return;
    const options = normalizeOptions(question);
    options.push({ text: "", checked: false });
    syncQuestionOptions(index, question, options);
  }

  function removeOption(index: number, optionIndex: number) {
    const question = editingExam?.context?.questions?.[index];
    if (!question) return;
    const options = normalizeOptions(question).filter((_, currentIndex) => currentIndex !== optionIndex);
    syncQuestionOptions(index, question, options);
  }

  function syncQuestionOptions(index: number, question: ExamQuestion, options: OptionItem[]) {
    const context = { ...(question.context ?? {}) };
    context.options = options.map((option) => ({
      text: option.text,
      checked: Boolean(option.checked),
    }));
    delete context.answer;
    updateQuestion(index, { context });
  }

  function importQuestions(ids: number[]) {
    if (!editingExam) return;
    const imported = bankQuestions
      .filter((question) => ids.includes(question.id))
      .map((question) => sanitizeTheoryQuestion(question));
    const questions = orderExamQuestions([...(editingExam.context?.questions ?? []), ...imported]);
    setEditingExam({
      ...editingExam,
      context: {
        ...editingExam.context,
        questions,
      },
      total_score: sumQuestionPoints(questions),
    });
    setImportVisible(false);
  }

  if (loading) {
    return (
      <div style={{ textAlign: "center", padding: "48px 0" }}>
        <Spin size="large" />
      </div>
    );
  }
  if (!currentExam) {
    return <Empty description="试卷不存在" />;
  }
 
  return (
    <>
      <Card
        className="admin-table-card"
        variant="borderless"
        title={
          <div style={{ textAlign: "center" }}>
            <h2 style={{ margin: "16px 0", fontSize: 24 }}>
              {editMode ? editingExam?.title : exam?.title}
            </h2>
            <Space size="large">
              <span>总分：{sumQuestionPoints(examQuestions)}分</span>
              <span>题目数：{examQuestions.length}题</span>
              <Tag color={currentExam.is_active ? "success" : "default"}>
                {currentExam.is_active ? "启用" : "禁用"}
              </Tag>
            </Space>
          </div>
        }
        extra={
          editMode ? (
            <Space wrap>
              <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>
                从题库导入
              </Button>
              <Button icon={<PlusOutlined />} onClick={openCreateQuestion}>
                手动添加题目
              </Button>
              <Button type="primary" onClick={() => void saveEdit()}>
                保存
              </Button>
              <Button onClick={cancelEdit}>取消</Button>
            </Space>
          ) : (
            <Space wrap>
              <Button onClick={() => navigate(backTo)}>返回列表</Button>
              <Button icon={<EditOutlined />} onClick={enterEditMode}>
                编辑试卷
              </Button>
              <Button danger icon={<DeleteOutlined />} onClick={handleDelete}>
                删除试卷
              </Button>
            </Space>
          )
        }
      >
      {editMode ? (
        <Form form={examForm} layout="vertical" style={{ marginBottom: 20 }}>
          <Form.Item
            name="title"
            label="试卷标题"
            rules={[{ required: true, message: "请输入试卷标题" }]}
          >
            <Input />
          </Form.Item>
          <Space size={16} align="start" wrap>
            <Form.Item name="is_active" label="启用状态" rules={[{ required: true }]}>
              <Radio.Group>
                <Radio value>启用</Radio>
                <Radio value={false}>禁用</Radio>
              </Radio.Group>
            </Form.Item>
            <Form.Item name="time_limit" label="时间限制(秒)">
              <InputNumber min={0} style={{ width: 180 }} />
            </Form.Item>
          </Space>
        </Form>
      ) : null}

      {examQuestions.length === 0 ? <Empty description="试卷中没有题目" /> : null}
      {examQuestions.map((question, index) => (
        <div
          key={`${question.id ?? "manual"}-${index}`}
          style={{
            marginBottom: 20,
            padding: 16,
            background: "#fafafa",
            borderRadius: 4,
            border: "1px solid #eee",
          }}
        >
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              gap: 12,
              marginBottom: 12,
            }}
          >
            <div style={{ flex: 1 }}>
              {editMode ? (
                <Space direction="vertical" size={12} style={{ width: "100%" }}>
                  <Space align="start" style={{ width: "100%", justifyContent: "space-between" }}>
                    <Space align="center" wrap>
                      <Typography.Text strong>{index + 1}.</Typography.Text>
                      <Select
                        value={question.question_type}
                        style={{ width: 120 }}
                        options={Object.entries(typeNames).map(([value, label]) => ({ value, label }))}
                        onChange={(value) => updateQuestionType(index, value)}
                      />
                      <Input
                        value={question.question}
                        onChange={(event) => updateQuestion(index, { question: event.target.value })}
                        style={{ width: 680, maxWidth: "100%" }}
                      />
                    </Space>
                    <Space wrap>
                      <Button
                        size="small"
                        icon={<ArrowUpOutlined />}
                        onClick={() => moveQuestion(index, -1)}
                        disabled={index === 0}
                      >
                        上移
                      </Button>
                      <Button
                        size="small"
                        icon={<ArrowDownOutlined />}
                        onClick={() => moveQuestion(index, 1)}
                        disabled={index === examQuestions.length - 1}
                      >
                        下移
                      </Button>
                      <Space size={6}>
                        <Typography.Text>分值:</Typography.Text>
                        <InputNumber
                          min={1}
                          value={question.points}
                          onChange={(value) => updateQuestion(index, { points: Number(value ?? 0) })}
                          style={{ width: 88 }}
                        />
                      </Space>
                      <Button danger size="small" icon={<DeleteOutlined />} onClick={() => removeQuestion(index)}>
                        删除
                      </Button>
                    </Space>
                  </Space>
                </Space>
              ) : (
                <>
                  <span style={{ fontWeight: 700, marginRight: 8 }}>{index + 1}.</span>
                  <span>{question.question}</span>
                  <Space style={{ marginLeft: 12 }}>
                    <Tag color={typeColors[question.question_type]}>
                      {typeNames[question.question_type]}
                    </Tag>
                    <Tag>{question.points}分</Tag>
                  </Space>
                </>
              )}
            </div>
          </div>
          {editMode ? (
            <InlineQuestionEditor
              question={question}
              onQuestionChange={(patch) => updateQuestion(index, patch)}
              onOptionChange={(optionIndex, label) => updateOption(index, optionIndex, label)}
              onOptionToggle={(optionIndex, checked) => toggleOptionCorrect(index, optionIndex, checked)}
              onAddOption={() => addOption(index)}
              onRemoveOption={(optionIndex) => removeOption(index, optionIndex)}
            />
          ) : (
            renderQuestionDetail(question)
          )}
        </div>
      ))}
      </Card>

      <Modal
        title="从题库导入题目"
        open={importVisible}
        onCancel={() => {
          setImportVisible(false);
          setSearchText("");
          setFilterType("all");
        }}
        footer={null}
        width={1000}
      >
        <Space style={{ marginBottom: 16, width: "100%" }} direction="vertical">
          <Space wrap>
            <Input.Search
              value={searchText}
              onChange={(event) => setSearchText(event.target.value)}
              placeholder="搜索题目内容"
              style={{ width: 320 }}
            />
            <Radio.Group
              value={filterType}
              onChange={(event) => setFilterType(event.target.value)}
              buttonStyle="solid"
            >
              <Radio.Button value="all">全部</Radio.Button>
              <Radio.Button value="single">单选题</Radio.Button>
              <Radio.Button value="multiple">多选题</Radio.Button>
              <Radio.Button value="judge">判断题</Radio.Button>
              <Radio.Button value="text">问答题</Radio.Button>
            </Radio.Group>
          </Space>
        </Space>
        <SelectableQuestionTable questions={filteredBankQuestions} onConfirm={importQuestions} />
      </Modal>

    </>
  );
}

function InlineQuestionEditor({
  question,
  onQuestionChange,
  onOptionChange,
  onOptionToggle,
  onAddOption,
  onRemoveOption,
}: {
  question: ExamQuestion;
  onQuestionChange: (patch: Partial<ExamQuestion>) => void;
  onOptionChange: (optionIndex: number, label: string) => void;
  onOptionToggle: (optionIndex: number, checked: boolean) => void;
  onAddOption: () => void;
  onRemoveOption: (optionIndex: number) => void;
}) {
  const type = question.question_type;
  const options = normalizeOptions(question);
  return (
    <div>
      {type === "single" || type === "multiple" ? (
        <Space direction="vertical" size={12} style={{ width: "100%" }}>
          {options.map((option, index) => (
            <Space key={`${index}-${option.text}`} style={{ display: "flex", width: "100%" }} align="center">
              <Typography.Text style={{ width: 20 }}>{String.fromCharCode(65 + index)}.</Typography.Text>
              <Input
                value={option.text}
                onChange={(event) => onOptionChange(index, event.target.value)}
                style={{ flex: 1 }}
              />
              <Space size={6}>
                <Typography.Text type="secondary">正确答案:</Typography.Text>
                <Switch
                  checked={Boolean(option.checked)}
                  checkedChildren="是"
                  unCheckedChildren="否"
                  onChange={(checked) => onOptionToggle(index, checked)}
                />
              </Space>
              {options.length > 2 ? (
                <Button danger type="text" icon={<DeleteOutlined />} onClick={() => onRemoveOption(index)} />
              ) : null}
            </Space>
          ))}
          <Button type="dashed" icon={<PlusOutlined />} onClick={onAddOption}>
            添加选项
          </Button>
        </Space>
      ) : null}

      {type === "judge" ? (
        <div style={{ paddingLeft: 24 }}>
          <Radio.Group>
            <Radio
              checked={Boolean(question.context?.answer)}
              onChange={() => onQuestionChange({ context: { ...(question.context ?? {}), answer: true } })}
            >
              正确
            </Radio>
            <Radio
              checked={question.context?.answer === false}
              onChange={() => onQuestionChange({ context: { ...(question.context ?? {}), answer: false } })}
            >
              错误
            </Radio>
          </Radio.Group>
        </div>
      ) : null}

      {type === "text" ? (
        <Input.TextArea
          rows={4}
          value={String(question.context?.answer ?? "")}
          onChange={(event) =>
            onQuestionChange({ context: { ...(question.context ?? {}), answer: event.target.value } })
          }
          placeholder="参考答案"
        />
      ) : null}
    </div>
  );
}

function SelectableQuestionTable({
  questions,
  onConfirm,
}: {
  questions: TheoryQuestion[];
  onConfirm: (ids: number[]) => void;
}) {
  const [picked, setPicked] = useState<number[]>([]);
  return (
    <>
      <Table
        rowSelection={{
          selectedRowKeys: picked,
          onChange: (keys) => setPicked(keys.map(Number)),
        }}
        columns={[
          { title: "ID", dataIndex: "id", width: 70 },
          { title: "题目", dataIndex: "question", ellipsis: true },
          {
            title: "类型",
            dataIndex: "question_type",
            width: 100,
            render: (value: QuestionType) => <Tag color={typeColors[value]}>{typeNames[value]}</Tag>,
          },
          { title: "分值", dataIndex: "points", width: 70 },
        ]}
        dataSource={questions}
        rowKey="id"
        pagination={{ pageSize: 8 }}
        size="small"
      />
      <div style={{ marginTop: 16, textAlign: "right" }}>
        <Button type="primary" disabled={picked.length === 0} onClick={() => onConfirm(picked)}>
          导入选中题目
        </Button>
      </div>
    </>
  );
}

function renderQuestionDetail(question: ExamQuestion) {
  const context = question.context ?? {};
  if (question.question_type === "judge") {
    return (
      <div style={{ paddingLeft: 24, color: "#666" }}>
        答案：{formatJudgeAnswer(context.answer)}
      </div>
    );
  }
  if (question.question_type === "text") {
    return (
      <div style={{ paddingLeft: 24, color: "#666" }}>
        参考答案：{String(context.answer ?? "-")}
      </div>
    );
  }
  const options = normalizeOptions(question);
  return (
    <div style={{ paddingLeft: 24 }}>
      {options.map((option, index) => (
        <div
          key={`${question.id ?? "manual"}-${index}`}
          style={{ marginBottom: 8, color: option.checked ? "#52c41a" : "#666" }}
        >
          {String.fromCharCode(65 + index)}. {option.text} {option.checked ? "✓" : ""}
        </div>
      ))}
    </div>
  );
}

function normalizeExamRecord(exam: ExamRecord, bank: TheoryQuestion[]) {
  const context = exam.context ?? {};
  const byId = new Map(bank.map((item) => [item.id, item]));
  let questions = Array.isArray(context.questions) ? context.questions.map(sanitizeTheoryQuestion) : [];
  if (!questions.length && Array.isArray(context.question_ids)) {
    questions = context.question_ids
      .map((id) => byId.get(Number(id)))
      .filter(Boolean)
      .map((item) => sanitizeTheoryQuestion(item as TheoryQuestion));
  }
  return {
    ...exam,
    context: {
      ...context,
      questions: orderExamQuestions(questions),
    },
  };
}

function sanitizeTheoryQuestion(question: TheoryQuestion | ExamQuestion) {
  return {
    id: question.id,
    question: question.question,
    question_type: question.question_type,
    points: question.points,
    context: { ...(question.context ?? {}) },
  };
}

function deepCloneExam(exam: ExamRecord): ExamRecord {
  return JSON.parse(JSON.stringify(exam)) as ExamRecord;
}

function orderExamQuestions<T extends { question_type: QuestionType }>(questions: T[]) {
  return [...questions].sort((a, b) => {
    const aIndex = questionTypeOrder.indexOf(a.question_type);
    const bIndex = questionTypeOrder.indexOf(b.question_type);
    if (aIndex !== bIndex) {
      return aIndex - bIndex;
    }
    return 0;
  });
}

function normalizeRows<T>(result: unknown): T[] {
  if (Array.isArray(result)) return result as T[];
  const object = (result ?? {}) as { data?: T[]; items?: T[]; list?: T[] };
  return object.data ?? object.items ?? object.list ?? [];
}

function normalizeCompetitionExam(result: unknown): ExamRecord | null {
  if (!isRecord(result)) {
    return null;
  }
  if ("exam" in result) {
    return isExamRecord(result.exam) ? result.exam : null;
  }
  return isExamRecord(result) ? result : null;
}

function isExamRecord(value: unknown): value is ExamRecord {
  return isRecord(value) && typeof value.id === "number";
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value));
}

function formatValue(value?: string) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}

function sumQuestionPoints(questions: ExamQuestion[]) {
  return questions.reduce((sum, question) => sum + Number(question.points ?? 0), 0);
}

function normalizeOptions(question: ExamQuestion): OptionItem[] {
  const rawOptions = Array.isArray(question.context?.options) ? question.context?.options : [];
  if (rawOptions.length) {
    return rawOptions.map((option: any) => ({
      text: String(option.text ?? ""),
      checked: Boolean(option.checked),
    }));
  }
  return [];
}

function createEmptyExamQuestion(): ExamQuestion {
  return {
    question: "",
    question_type: "single",
    points: 5,
    context: {
      options: [
        { text: "", checked: false },
        { text: "", checked: false },
      ],
    },
  };
}

function createQuestionContextByType(type: QuestionType) {
  if (type === "single" || type === "multiple") {
    return {
      options: [
        { text: "", checked: false },
        { text: "", checked: false },
      ],
    };
  }
  if (type === "judge") {
    return { answer: true };
  }
  return { answer: "" };
}

function formatJudgeAnswer(value: unknown) {
  if (value === true || value === "true" || value === 1 || value === "1") {
    return "正确";
  }
  if (value === false || value === "false" || value === 0 || value === "0") {
    return "错误";
  }
  return "-";
}
