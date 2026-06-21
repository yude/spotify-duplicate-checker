# spotify-duplicate-checker

公開 Spotify プレイリスト内で、異なるアルバムに含まれる同一曲名のトラックを検出する CLI ツール。

## 必要条件

- Go 1.26+
- Spotify アプリの Client ID と Client Secret
  - [Spotify Developer Dashboard](https://developer.spotify.com/dashboard) で作成

## インストール

```bash
go build -o spotify-duplicate-checker .
```

## 使い方

```bash
export SPOTIFY_CLIENT_ID="your-client-id"
export SPOTIFY_CLIENT_SECRET="your-client-secret"

./spotify-duplicate-checker <playlist-id-or-url>
```

例:

```bash
./spotify-duplicate-checker "https://open.spotify.com/playlist/5OXSszQKe9OCYHSlrJ8Cmb"
./spotify-duplicate-checker "5OXSszQKe9OCYHSlrJ8Cmb"
```

## 出力例

```
fetched 245 tracks

found 2 track name(s) appearing on different albums:

  "Hoshikuzu Serenade"
    - album: "STAR TRAVELERS"  artist: FRUITS ZIPPER  id: 6x4wR1mh4bYrOhl6kIwyBn
    - album: "NEW KAWAII"  artist: FRUITS ZIPPER  id: 3vY5lRqG5mRJhqLmNFZN3S

  "Watashi no Ichiban Kawaiitokoro"
    - album: "Watashi no Ichiban Kawaiitokoro"  artist: FRUITS ZIPPER  id: 7hJpKTl4pQ6q0uP8vXq5jF
    - album: "NEW KAWAII"  artist: FRUITS ZIPPER  id: 2kLmXqMZY9QpXKjGvN8vRr
```

## 認証について

ユーザーログイン不要の Client Credentials フローを使用しているため、公開プレイリストであれば Client ID / Client Secret のみで動作する。
