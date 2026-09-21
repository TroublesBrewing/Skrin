// Package version holds Skrin's release number. New features bump the
// minor number and fixes the patch number; see CHANGELOG.md.
package version

// Version is the current release, without the leading "v".
const Version = "0.47.0"

// Beta says whether this build offers beta features: experiments that are
// built, switched on by hand and tried, without being part of what Skrin
// promises. They are off unless asked for twice — Settings has to be in
// beta mode, and the feature itself switched on.
//
// Setting this to false is how a release leaves every experiment out: the
// beta block disappears from Settings and each beta feature reads as off,
// whatever config.toml says. v1.0 ships with it false.
const Beta = true
