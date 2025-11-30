# Quick Command Reference

## 🚀 Start Commands

```bash
# Local MongoDB
./scripts/run-local.sh api start

# Cloud MongoDB Atlas
./scripts/run-dev.sh api start

# Alternative: Environment variable
GEGO_ENV=local gego api start
GEGO_ENV=dev MONGODB_CLOUD_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/" gego api start
```

## 🔧 Setup Commands

```bash
# Initialize configuration
GEGO_ENV=local gego init

# Add OpenAI LLM
gego llm add openai --api-key sk-xxx --model gpt-4

# Add Anthropic LLM
gego llm add anthropic --api-key sk-ant-xxx --model claude-3-opus-20240229

# List all LLMs
gego llm list
```

## 📝 Prompt Commands

```bash
# Create prompt (interactive)
gego prompt create

# Create prompt (direct)
gego prompt create --template "What are the best {category} brands?"

# List prompts
gego prompt list

# View prompt details
gego prompt get <prompt-id>

# Delete prompt
gego prompt delete <prompt-id>
```

## ⏰ Schedule Commands

```bash
# Create schedule (interactive)
gego schedule add

# List schedules
gego schedule list

# Run scheduler
gego scheduler run --schedule-id <id>

# Run schedule once
gego schedule run <schedule-id>
```

## 📊 Stats & Analytics

```bash
# View keyword stats
gego stats keywords

# View prompt stats
gego stats prompts

# View LLM stats
gego stats llms

# Search responses
gego search --keyword "Netflix"
```

## 🌍 Your MongoDB Info

**IP Address:** `106.222.202.9`
**Cloud URI:** `mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/`

## 🔗 Useful Links

- [MongoDB Atlas Dashboard](https://cloud.mongodb.com/)
- [Full Setup Guide](MONGODB_SETUP.md)
- [Environment Configuration](docs/ENVIRONMENT_SETUP.md)

