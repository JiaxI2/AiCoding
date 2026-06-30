# Safe Rewrite Plan Rule

A safe rewrite plan is required when:

- a command contains Bash aliases;
- a command contains destructive verbs;
- a command contains `&&`, `||`, multiple `;`, redirection, or long chained pipelines;
- a command touches registry, services, network, ACL, boot, partitions, or system directories.

The plan must include:

1. original command;
2. detected risks;
3. PowerShell-native replacement;
4. validation commands;
5. rollback/backup step when needed;
6. block decision.

The rewrite plan must not execute the command.
