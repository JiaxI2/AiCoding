# Repair Loop Policy

The repair loop is an AI-assisted patch workflow, not an autonomous deployment system.

## Allowed by default

- Read logs.
- Edit allowed source paths.
- Run build command.
- Run test command.
- Record attempts.
- Export context.

## Disallowed by default

- Flash.
- Reset.
- Halt.
- Open-loop hardware execution.
- High-power motor operation.
- Commit.
- Push.
- Edit forbidden paths.
- Edit tests to force pass.

## PASS definition

Only the configured test runner can produce PASS.
