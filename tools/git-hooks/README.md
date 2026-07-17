# Git hooks

`prepare-commit-msg` appends the project's co-author trailers to every commit,
so both show up as contributors on GitHub:

```
Co-Authored-By: Claude <noreply@anthropic.com>
Co-Authored-By: Codex <security@openai.com>
```

It is idempotent (keyed on the co-author email), so a trailer an agent already
added by hand is never duplicated.

Activate it once per clone:

```sh
git config core.hooksPath tools/git-hooks
```

`deploy-soloagency.sh` appends the same trailers to its auto-generated commit
messages, so deploy commits carry them even in a clone where the hook path is
not set.
