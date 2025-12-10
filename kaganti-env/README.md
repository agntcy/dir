# Deploy from scratch
```bash
GITHUB_USER=YOUR_GH_USER GITHUB_TOKEN=YOUR_GH_PAT_TOKEN GITHUB_EMAIL=YOUR_GH_EMAIL task deploy
```

# Clean up
```bash
task clean
```

# Port-forward the weather service
```bash
task kagenti:weather-service:port-forward
```

# Send a message to the weather
```bash
MESSAGE="What is the weather in Antarctica" task kagenti:weather-service:call
```
or use default message
```bash
task kagenti:weather-service:call
```
