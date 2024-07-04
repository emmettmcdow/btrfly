# Contributing
## Requirements
- Go (>= 1.21.0)
- Docker
- A Unix or Linux machine(not strictly necessary but highly recommended)

## Learning GO Resources
- https://go.dev/tour/welcome/1
- https://go.dev/doc/code
- https://go.dev/ref/spec
- https://go.dev/doc/effective_go

## Testing
We want to aim for 90% or higher test coverage. With every piece of code you add, make sure you add
corresponding unit tests. We don't really care about the distinction between unit/integration tests.
We really just want tests that:
1. Are robust (a refactor will not break them much).
2. Run as quickly as possible.
3. Effectively test the intended behavior. 
4. Avoid using complicated "Mocking"

To see some of my inspiration in my approach to testing, see [Mitchell Hashimoto's talk on Advanced
Golang Testing](https://www.youtube.com/watch?v=8hQG7QlcLBk)

## Committing
We want our commits to be as small as is reasonable. We want our commit names to be short but 
descriptive. See [this](https://cbea.ms/git-commit/) (canonical imo) guide to learn more.

## Formatting
Go is awesome. Prior to submitting a PR, please run `gofmt -w .` on the codebase. This will auto-
format your code.

