// Command space-sim starts the interactive Space Sim application.
//
// It parses CLI flags, loads persisted application configuration, constructs
// the app runtime, and then runs either the interactive renderer or automated
// performance mode. Most application behavior lives in internal/space/app and
// related internal packages; this command stays intentionally thin.
package main
