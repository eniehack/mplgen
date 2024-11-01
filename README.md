# monthplgen

ListenBrainzに記録から今月聞いた曲を取得するCLIツール

## install

### requires

- golang
- listenbrainz account

### instructions

1. `git clone` & cd
2. go build

## option

- month: 収集する月
  - e.g. `2023-07`
- timezone: タイムゾーン。月の初めと終わりの時間を特定するのに使います
  - default: `utc`
- user: Listenbrainzのアカウント名
  - e.g. `eniehack`

### 例

```
$ ./monthplgen -month 2024-10 -timezone Asia/Tokyo -user eniehack
```
