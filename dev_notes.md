# Development Notes

## To-Do

- [x] UI: Embed actions
- [x] Change the cache file name to UUID
- [x] Caching logic
- [x] Create functions to interact with Pocket Casts API
    - Convert retrieved data into already defined data structure
        - Queue
        - Podcasts
        - Podcast episodes (a method of `Podcast` object)
        - Latest episodes
    - Actions
        - Log in and store token
        - Sync state with Pocket Casts
        - Add episode to queue / archive
        - Subscribe / unsubscribe
- [x] Adapt existing objects to the changes
    - Add `uuid` field to `Episode` and `Podcast` objects

## Pocket Casts API

### Log in

```shell
curl https://api.pocketcasts.com/user/login \
    -d '{"email": "<EMAIL>", "password": "<PASSWORD>"}'
```

#### Response schema

```shell
{ token: string }
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

#### Episodes Info

```shell
curl https://podcast-api.pocketcasts.com/podcast/full/<PODCAST_UUID>
```

##### Response schema

```
{
    podcast: {
        episodes: [
            {
                uuid: string,
                title: string,
                url: string,
                published: string,
                duration: number,
                playedUpTo: number,
            }
        ],
        uuid: string,
        title: string,
        author: string,
        description: string,
    }
}
```
#### Episodes with show notes

```shell
curl https://podcast-api.pocketcasts.com/mobile/show_notes/full/<PODCAST_UUID>
```

##### Response schema

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
        ]
        uuid: string
    }
}
```

#### Podcast page

`https://play.pocketcasts.com/discover/podcast/<PODCAST_UUID>`

#### Podcast thumbnail

`https://static.pocketcasts.com/discover/images/webp/200/<PODCAST_UUID>.webp`

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

### Actions

#### Play next

```shell
curl https://api.pocketcasts.com/up_next/play_next \
    -d '{"version":2,"episode":{"uuid":"<UUID>","podcast":"<PODCAST_UUID>","title":"<EPISODE_TITLE>","url":"<EPISODE_URL>"}}'
```

#### Play last

```shell
curl https://api.pocketcasts.com/up_next/play_last \
    -d '{"version":2,"episode":{"uuid":"<UUID>","podcast":"<PODCAST_UUID>","title":"<EPISODE_TITLE>","url":"<EPISODE_URL>"}}'
```

#### Remove from queue

```shell
curl https://api.pocketcasts.com/up_next/remove \
    -d '{"version":2, "uuids":["<UUID>"]}'
```

#### Update episode

- Update playback position: `"position": "<position>", "status": 2`
- Mark as played: `"status": 3`

```shell
curl https://api.pocketcasts.com/sync/update_episode \
    -d '{"uuid":"<UUID>","podcast":"<PODCAST_UUID>","status":3}'
```

#### Archive episodes

```shell
curl https://api.pocketcasts.com/sync/update_episodes_archive \
    -d '{"episodes":[{"uuid":"<UUID>","podcast":"<PODCAST_UUID>"}],"archive":true}'
```

#### Subscribe

```shell
curl https://api.pocketcasts.com/user/podcast/subscribe \
    -d '{"uuid":"<PODCAST_UUID>"}'
```

#### Unsubscribe

```shell
curl https://api.pocketcasts.com/user/podcast/unsubscribe \
    -d '{"uuid":"<PODCAST_UUID>"}'
```

