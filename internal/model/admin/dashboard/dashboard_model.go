// Package dashboard defines aggregated Admin dashboard data.
package dashboard

import "time"

type Summary struct {
	GeneratedAt     time.Time      `json:"generated_at"`
	Users           UserMetrics    `json:"users"`
	QuestionBank    ContentMetrics `json:"question_bank"`
	LearningModules ModuleMetrics  `json:"learning_modules"`
	QuizCompletion  QuizCompletion `json:"quiz_completion"`
	UjiKomToday     int64          `json:"uji_kom_today"`
	Tasks           TaskMetrics    `json:"tasks"`
}

type SummaryPeriods struct {
	MonthStart         time.Time
	MonthEnd           time.Time
	PreviousMonthStart time.Time
	WeekStart          time.Time
	QuizWindowStart    time.Time
	TodayStart         time.Time
	TomorrowStart      time.Time
}

type UserMetrics struct {
	Total              int64      `json:"total"`
	ByRole             RoleCounts `json:"by_role"`
	NewThisMonth       int64      `json:"new_this_month"`
	NewLastMonth       int64      `json:"new_last_month"`
	MonthGrowthPercent *float64   `json:"month_growth_percent"`
}

type RoleCounts struct {
	Students int64 `json:"students"`
	Tutors   int64 `json:"tutors"`
	Admins   int64 `json:"admins"`
}

type ContentMetrics struct {
	Total       int64               `json:"total"`
	NewThisWeek int64               `json:"new_this_week"`
	ByLevel     []LevelContentCount `json:"by_level"`
}

type LevelContentCount struct {
	EducationLevel EducationLevel `json:"education_level"`
	Total          int64          `json:"total"`
	NewThisWeek    int64          `json:"new_this_week"`
}

type ModuleMetrics struct {
	Total             int64              `json:"total"`
	Published         int64              `json:"published"`
	PublishedThisWeek int64              `json:"published_this_week"`
	ByLevel           []LevelModuleCount `json:"by_level"`
}

type LevelModuleCount struct {
	EducationLevel    EducationLevel `json:"education_level"`
	Total             int64          `json:"total"`
	Published         int64          `json:"published"`
	PublishedThisWeek int64          `json:"published_this_week"`
}

type EducationLevel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type QuizCompletion struct {
	PeriodStart  string   `json:"period_start"`
	PeriodEnd    string   `json:"period_end"`
	Completed    int64    `json:"completed"`
	Total        int64    `json:"total"`
	NotCompleted int64    `json:"not_completed"`
	Percent      *float64 `json:"percent"`
}

type TaskMetrics struct {
	PendingCount   int64          `json:"pending_count"`
	Items          []TaskItem     `json:"items"`
	RecentActivity []TaskActivity `json:"recent_activity"`
}

type TaskItem struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Status    string    `json:"status"`
	CreatedBy TaskActor `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type TaskActivity struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Status    string    `json:"status"`
	UpdatedBy TaskActor `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TaskActor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TrendRow struct {
	BucketStart    time.Time
	EducationLevel EducationLevel
	SortOrder      int
	Sessions       int64
	Average        *float64
}

type DailyQuizTrend struct {
	Period      string        `json:"period"`
	PeriodStart string        `json:"period_start"`
	PeriodEnd   string        `json:"period_end"`
	Buckets     []string      `json:"buckets"`
	Series      []TrendSeries `json:"series"`
}

type SearchResults struct {
	Users           []UserSearchResult           `json:"users"`
	Questions       []QuestionSearchResult       `json:"questions"`
	LearningModules []LearningModuleSearchResult `json:"learning_modules"`
	Quizzes         []QuizSearchResult           `json:"quizzes"`
}

type UserSearchResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type QuestionSearchResult struct {
	ID             string         `json:"id"`
	Code           string         `json:"code"`
	Preview        string         `json:"preview"`
	Type           string         `json:"type"`
	EducationLevel EducationLevel `json:"education_level"`
	Grade          EducationLevel `json:"grade"`
}

type LearningModuleSearchResult struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Status         string         `json:"status"`
	EducationLevel EducationLevel `json:"education_level"`
	Grade          EducationLevel `json:"grade"`
}

type QuizSearchResult struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Status         string         `json:"status"`
	StartAt        *time.Time     `json:"start_at"`
	EndAt          *time.Time     `json:"end_at"`
	EducationLevel EducationLevel `json:"education_level"`
	Grade          EducationLevel `json:"grade"`
}

type TrendSeries struct {
	EducationLevel EducationLevel `json:"education_level"`
	Points         []TrendPoint   `json:"points"`
}

type TrendPoint struct {
	PeriodStart string   `json:"period_start"`
	Sessions    int64    `json:"sessions"`
	Average     *float64 `json:"average"`
}
