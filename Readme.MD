# fly-notification-agent
Notify webhooks based on vm state

# supported states
- machine_started
- machine_stopped
- oom

# payload structure
- type
- data
    - machine_id
    - app_name
    - state
    - at

# usage
drop this line in your dockerfile 
```
curl -L https://fly-notification-agent.fly.dev/install.sh | sh
```

add this to your machine config
```
{
    "name": "events",
    "entrypoint": ["/bin/bash", "-c"],
    "cmd": ["~/.fly/fly-notification-agent", "--webhook", "<insert-your-webhook-url>]
}
```