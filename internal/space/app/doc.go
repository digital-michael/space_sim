// Package app coordinates the Space Sim application's runtime lifecycle.
//
// It validates configuration, manages persisted window and session settings,
// wires simulation state to renderer state, and owns the interactive and
// performance-mode execution flows used by the main command.
package app