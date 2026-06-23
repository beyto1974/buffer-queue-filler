## README

Go tool that fills Buffer queues from drafts. It fetches all channels for a Buffer organization, checks each queue size, optionally shuffles drafts, and moves drafts into the queue (automatic scheduling) until `--maxQueueSize` is reached. Buffer API requests use a Bearer token in the `Authorization` header, and you need a Buffer API key plus an organization ID. [developers.buffer](https://developers.buffer.com/guides/getting-started.html)

### Usage

```bash
go run . --maxQueueSize=9 --bufferOrgId=ORG_ID --bufferKey=BUFFER_API_KEY --shuffleDrafts
```

### Flags

- `--maxQueueSize`: Maximum queued posts per channel before stopping.
- `--bufferOrgId`: Buffer organization ID.
- `--bufferKey`: Buffer API key.
- `--shuffleDrafts`: Shuffle drafts before queueing. [buffer](https://buffer.com/api)

### Behavior

- Lists all channels in the target Buffer organization.
- Reads queued posts for each channel.
- If queue size is below the limit, reads drafts for that channel.
- Optionally shuffles drafts.
- Pushes drafts into the queue until the limit is reached. [developers.buffer](https://developers.buffer.com/examples/get-organizations.html)

## License

This project is licensed under the MIT License.