# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-10-18

### Added

- AppStream metadata in the `.deb` and `.rpm` packages: once installed, KDE Discover shows Zenith with its name, icon, developer, description and license instead of an anonymous package.
- The application icon is also installed in the standard `hicolor` icon theme.

## [1.2.0] - 2026-10-18

### Added

- Fedora package (`zenith-<version>-1.x86_64.rpm`, Fedora 40 or later) attached to every release; dnf installs the GTK/WebKitGTK dependencies.
- Automatic updates for Zenith installed from the `.rpm`: the app downloads the new package, verifies it, installs it with dnf after asking for the password, and restarts.
- The project is now released under the MIT license.

## [1.1.0] - 2026-10-18

### Added

- The "New task" dialog now has a "Column" selector above the title, so a task can be created directly in any column instead of always in the first one. The first column is preselected.
