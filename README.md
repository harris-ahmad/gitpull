# gitpull

A command-line tool for managing multiple Git repositories at once. Run it from a directory containing several repos and it operates across all of them. Run it inside a single repo and it behaves like a standard Git workflow tool.

## Installation

### From source

Requires Go 1.21 or later.

```bash
git clone https://github.com/harris-ahmad/gitpull.git
cd gitpull
go build -o gitpull .
mv gitpull /usr/local/bin/
```

### Homebrew

```bash
brew tap harris-ahmad/tap
brew install gitpull
```

## Configuration

On first run, `gitpull` works with no configuration. To persist settings, create `~/.gitpullrc`:

```
token=your_github_token
parallel=8
ssh=true
ollama_url=http://localhost:11434
ollama_model=mistral
```

| Key | Default | Description |
|-----|---------|-------------|
| `token` | `$GITHUB_TOKEN` | GitHub personal access token |
| `parallel` | `4` | Number of parallel clone workers |
| `ssh` | `false` | Use SSH URLs for cloning |
| `ollama_url` | `http://localhost:11434` | Ollama server URL |
| `ollama_model` | `mistral` | Model to use for AI features |

## Commands

### Clone

Clone all repositories for a GitHub user or organization.

```bash
gitpull clone <username>
gitpull clone <username> --org <org>
gitpull clone <username> --include-private
gitpull clone <username> --ssh
gitpull clone <username> --skip-forks
gitpull clone <username> --skip-archived
```

### Sync

Fetch and pull all repositories in the current directory. Skips repos with local changes or predicted merge conflicts.

```bash
gitpull sync
gitpull sync --report       # dry-run, no actual pulls
gitpull sync --ai           # AI summary of local changes and conflicts
```

### Status

Show the dirty or clean state of all repositories.

```bash
gitpull status
```

### Diff

Show staged and unstaged changes across all repositories, paged through `less`.

```bash
gitpull diff
```

### Commit

Generate an AI commit message from the current diff, review it, then add, commit, and push.

```bash
gitpull commit
```

You will be prompted to accept (`y`), edit (`e`), or reject (`n`) the suggested message.

### Branch

List, create, switch, or delete branches across all repositories.

```bash
gitpull branch                    # list current branch for each repo
gitpull branch <name>             # create a new branch
gitpull branch --switch <name>    # switch to an existing branch
gitpull branch --delete <name>    # delete a branch
```

### Stash

Stash or restore local changes across all repositories.

```bash
gitpull stash push
gitpull stash push --message "wip: feature work"
gitpull stash pop
```

### Clean

Show or remove untracked files across all repositories. Dry-run by default.

```bash
gitpull clean
gitpull clean --force
```

### Standup

Generate a standup summary from recent commits using a local AI model.

```bash
gitpull standup
gitpull standup --since "48 hours ago"
```

Requires [Ollama](https://ollama.com) to be running locally.

### Ask

Ask a natural language question about your repositories.

```bash
gitpull ask "what was I working on yesterday?"
gitpull ask "which repos have uncommitted changes?"
gitpull ask "which repos are behind their remote?"
```

Supports follow-up questions — context is retained within the session. Requires Ollama.

## AI Features

The AI commands (`sync --ai`, `commit`, `standup`, `ask`) use a locally running [Ollama](https://ollama.com) instance. No data is sent to external servers.

```bash
# Install Ollama, then pull a model
ollama pull mistral
ollama serve
```

Any model supported by Ollama works. Larger models produce better results at the cost of speed.

## License

MIT
