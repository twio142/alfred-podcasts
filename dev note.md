# Development Note

## Features

- [ ] Create functions to interact with Pocket Casts API
    - Convert retrieved data into already defined data structure

## Pocket Casts API

### Log in

```shell
curl https://api.pocketcasts.com/user/login \
    -d '{"email": "<EMAIL>", "password": "<PASSWORD>"}'
```

### Queue

```shell
curl https://api.pocketcasts.com/up_next/list \
    -H "Authorization: Bearer <TOKEN>" \
    -d '{"version":2,"model":"webplayer","serverModified":"<timestamp>","showPlayStatus":true}'
```

#### Response schema

```
{
    episodes: [
        {
            uuid: string,
            title: string,
            url: string,
            podcast: string,
            published: string,
        }
    ],
    episodeSync: [
        {
            uuid: string,
            duration: number,
            playedUpTo: number?,
        }
    ]
}
```

### Podcasts

```shell
curl https://api.pocketcasts.com/user/podcast/list -d '{"v":1}'
```

#### Response schema

```
{
    folders: [],
    podcasts: [
        {
            uuid: string,
            title: string,
            url: string,
            description: string,
            author: string,
            latestEpisodePublished: string,
        }
    ]
}
```

### Podcast episodes

```shell
curl https://api.pocketcasts.com/<PODCAST_UUID>/episodes_<TIMESTAMP>.json
```

#### Response schema

```
{
    podcast: {
        episodes: [
            {
                uuid: string,
                title: string,
                url: string,
                show_notes: string,
                published: string,
            }
        ],
        uuid: string
    }
}
```

### Latest

```shell
curl https://api.pocketcasts.com/user/new_releases -d '{}'
```

#### Response schema

```
{
    episodes: [
        {
            uuid: string,
            title: string,
            url: string,
            podcastTitle: string,
            podcastUuid: string,
            published: string,
            duration: number,
            playedUpTo: number,
        }
    ],
    total: number
}
```

### History

```shell
curl https://api.pocketcasts.com/user/history -d '{}'
```

#### Response schema

```
{
    episodes: [
        {
            uuid: string,
            title: string,
            url: string,
            podcastTitle: string,
            podcastUuid: string,
            published: string,
            duration: number,
            playedUpTo: number,
        }
    ],
    total: number
}
```

### Update episode

- Update payback position: `"position": "<position>", "status": 2`
- Mark as played: `"status": 3`

```shell
curl https://api.pocketcasts.com/sync/update_episode \
    -d '{"uuid":"<UUID>","podcast":"<PODCAST_UUID>","status":3}'
```

### Archive episodes

```shell
curl https://api.pocketcasts.com/sync/update_episodes_archive \
    -d '{"episodes":[{"uuid":"<UUID>","podcast":"<PODCAST_UUID>"}],"archive":true}'
```

