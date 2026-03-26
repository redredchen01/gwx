package cmd

// WorkflowCmd is the command group for gwx workflow subcommands.
type WorkflowCmd struct {
	WeeklyDigest     WeeklyDigestCmd     `cmd:"weekly-digest" help:"Weekly activity digest"`
	ContextBoost     ContextBoostCmd     `cmd:"context-boost" help:"Deep context gathering for a topic"`
	BugIntake        BugIntakeCmd        `cmd:"bug-intake" help:"Gather context for a bug report"`
	TestMatrix       TestMatrixCmd       `cmd:"test-matrix" help:"Manage test results in Sheets"`
	SpecHealth       SpecHealthCmd       `cmd:"spec-health" help:"Track spec status in Sheets"`
	SprintBoard      SprintBoardCmd      `cmd:"sprint-board" help:"Sprint board in Sheets"`
	ReviewNotify     ReviewNotifyCmd     `cmd:"review-notify" help:"Notify reviewers about a spec"`
	EmailFromDoc     EmailFromDocCmd     `cmd:"email-from-doc" help:"Send email from a Google Doc"`
	SheetToEmail     SheetToEmailCmd     `cmd:"sheet-to-email" help:"Send personalized emails from Sheet data"`
	ParallelSchedule ParallelScheduleCmd `cmd:"parallel-schedule" help:"Schedule parallel 1-on-1 reviews"`
	Digest           WeeklyDigestCmd     `cmd:"digest" help:"Alias for weekly-digest" hidden:""`
}
