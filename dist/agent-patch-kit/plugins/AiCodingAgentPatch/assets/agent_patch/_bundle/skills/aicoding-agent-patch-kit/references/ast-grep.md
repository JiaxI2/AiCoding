# ast-grep through apatch

Use `apatch ast` when changing code structure rather than literal text.

Example:

```powershell
apatch ast --lang ts --pattern '$A && $A()' --rewrite '$A?.()' --preview
apatch ast --lang ts --pattern '$A && $A()' --rewrite '$A?.()' --apply
```

Use language flags such as `c`, `cpp`, `ts`, `js`, `python`, or other ast-grep-supported language names.
