# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]
- Add last cycle and last cycle to current event parsers. [#32](https://github.com/xmidt-org/interpreter/pull/32)
- Add cycle validators to validate the order of events and that the latest online event is the result of a true reboot. [#33](https://github.com/xmidt-org/interpreter/pull/33)
- Change parsers to sort event list by newest to oldest. Add current cycle parser. [#34](https://github.com/xmidt-org/interpreter/pull/34)

## [v0.0.4]
- Add validator to validate consistent device id and enhancements to boot-time validator. Introduce tags and `TaggedError` interface. [#18](https://github.com/xmidt-org/interpreter/pull/18)
- Replace `MetricsLogError` with `TaggedError`. Add validator to futher validate birthdates and event-types. [#21](https://github.com/xmidt-org/interpreter/pull/21)
- Add `EventsParser` that returns a subset of a list of events. [#26](https://github.com/xmidt-org/interpreter/pull/26)
- Add `CycleValidator` to validate a list of events. [#29](https://github.com/xmidt-org/interpreter/pull/29)
- Remove Comparator from `EventFinder`. [#31](https://github.com/xmidt-org/interpreter/pull/31)

## [v0.0.3]
- Add labels for errors for prometheus error metrics logging. [#15](https://github.com/xmidt-org/interpreter/pull/15)
- Add function to get event-type. [#16](https://github.com/xmidt-org/interpreter/pull/16)

## [v0.0.2]
- Move initial code from `glaukos`. [#5](https://github.com/xmidt-org/interpreter/pull/5)
- Add `Comparator` interface to compare events. Update `DestinationValidator` to validate that an event destination matches the event regex. [#12](https://github.com/xmidt-org/interpreter/pull/12)

## [v0.0.1]
- Initial creation

[Unreleased]: https://github.com/xmidt-org/interpreter/compare/v0.0.4..HEAD
[v0.0.4]: https://github.com/xmidt-org/interpreter/compare/v0.0.3...v0.0.4
[v0.0.3]: https://github.com/xmidt-org/interpreter/compare/v0.0.2...v0.0.3
[v0.0.2]: https://github.com/xmidt-org/interpreter/compare/v0.0.1...v0.0.2
[v0.0.1]: https://github.com/xmidt-org/interpreter/compare/0.0.0...v0.0.1
