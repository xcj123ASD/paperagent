package model

type Journal struct {
	Name          string   `json:"name"`
	Abbr          string   `json:"abbr"`
	Discipline    string   `json:"discipline"`
	Sub           string   `json:"sub"`
	Level         string   `json:"level"`
	Columns       []string `json:"columns"`
	WordMin       int      `json:"word_min"`
	WordMax       int      `json:"word_max"`
	EmpiricalOnly bool     `json:"empirical_only"`
	TitleStyle    string   `json:"title_style"`
	Cycle         string   `json:"cycle"`
	Score         float64  `json:"score"`
	Breakdown     string   `json:"breakdown"`
}

type TopicCandidate struct {
	Title    string   `json:"title"`
	Disc     string   `json:"disc"`
	Keywords []string `json:"keywords"`
	Type_    string   `json:"type"`
	Score    float64  `json:"score"`
	Why      string   `json:"why"`
}

type SamplePaper struct {
	Title  string `json:"title"`
	Source string `json:"source"`
	Year   int    `json:"year"`
	Method string `json:"method"`
	URL    string `json:"url"`
}

type PolicySource struct {
	Name string `json:"name"`
	Org  string `json:"org"`
	Date string `json:"date"`
	URL  string `json:"url"`
}

type Check struct {
	Level string `json:"level"`
	Stage string `json:"stage"`
	Msg   string `json:"msg"`
}

type Step1Output struct {
	Discipline   string           `json:"discipline"`
	Topic        string           `json:"topic"`
	Keywords     []string         `json:"keywords"`
	ArticleType  string           `json:"article_type"`
	Journals     []Journal        `json:"journals"`
	Topics       []TopicCandidate `json:"topics"`
	SamplePapers []SamplePaper    `json:"sample_papers"`
	PolicySrcs   []PolicySource   `json:"policy_srcs"`
	Checks       []Check          `json:"checks"`
	TraceID      string           `json:"trace_id"`
	InputHash    string           `json:"input_hash"`
}

type Frame struct {
	Title           string     `json:"title"`
	TitleAlt        []string   `json:"title_alt"`
	AbstractCN      string     `json:"abstract_cn"`
	AbstractEN      string     `json:"abstract_en"`
	Keywords        []string   `json:"keywords"`
	Sections        []Section  `json:"sections"`
	WordCount       int        `json:"word_count"`
	TargetJournal   string     `json:"target_journal"`
	Column          string     `json:"column"`
	ArticleType     string     `json:"article_type"`
	FigureTablePlan []FigTable `json:"figure_table_plan"`
	RiskHints       []string   `json:"risk_hints"`
}

type Section struct {
	Name             string   `json:"name"`
	Role             string   `json:"role"`
	KeyPoints        []string `json:"key_points"`
	RequiredEvidence []string `json:"required_evidence"`
	WordBudget       int      `json:"word_budget"`
	MustCite         []string `json:"must_cite"`
	Forbidden        []string `json:"forbidden"`
	Content          string   `json:"content"` // M1: 正文，由 step3 填充
}

type FigTable struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	DataSource string `json:"data_source"`
	Note       string `json:"note"`
}