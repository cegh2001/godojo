package tui

// Command tests for the GoDojo TUI.
// Old command tests for runTestsCmd, fetchHintCmd, loadProgressCmd, and loadTopicExercisesCmd
// have been removed as those commands were eliminated in the TUI model simplification.
// The sensei communication is now handled through SenseiService.ProcessMessage,
// which returns results via toolStatusMsg and senseiResponseMsg channels.
