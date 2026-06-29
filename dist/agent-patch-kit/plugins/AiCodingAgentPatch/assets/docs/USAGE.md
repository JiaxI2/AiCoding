# Usage

## Literal replacement

```powershell
apatch status
apatch scan "old" --fixed
apatch replace --old "old" --new "new" --fixed --preview
apatch replace --old "old" --new "new" --fixed --apply
apatch verify --old "old" --new "new" --fixed
apatch summary
```

## Regex replacement

```powershell
apatch replace --old "v\d+\.\d+\.\d+" --new "v0.2.0" --regex --preview
```

Prefer `--fixed` unless regex is actually required.

## Structural code rewrite

```powershell
apatch ast --lang ts --pattern '$A && $A()' --rewrite '$A?.()' --preview
```

## Markdown link validation

```powershell
apatch links --mode offline --include-fragments full
```
