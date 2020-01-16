# youtubelivevote
Youtubeライブストリーミングのチャットで投票を行うためのツールです。

1択での得票計算から任意の個数の選択肢での投票に対応しています。

コマンド実行後の5秒後から指定した秒数の間のチャットメッセージのうち、選択肢の文字もしくは選択肢に振られた番号を意味する半角 1 ~ n が含まれているメッセージを投票とみなして集計します。（１アカウント一票で、一番最初のメッセージが有効になります）

集計途中でエンターキーを押すことで、投票を打ち切ることができます。

※配信中のチャットでのみ使用可能です。過去のチャットや、アーカイブのチャットに対しては使用できません。

# install
```
go get -u github.com/umineko1996/youtubelivevote
```

# 使い方

## 初回実行時
初回実行時はYoutubeAPIを実行するためにGoogle OAuth2ログインを求められます。

以下のエラーが発生する場合はログインするアカウントでYoutubeのライブストリーミングを有効にする必要があります。
```
"code": 403, "message": "The user is not enabled for live streaming."
```
https://www.youtube.com/features  
https://stackoverflow.com/questions/32362725/youtube-streaming-api-says-user-is-not-enabled-for-live-streaming

## usage

```
usage: youtubelivevote -id video_id [-t voting_time] [-s start_wait_time] [-m multiple_ballots] [choice1 choice2, ...]
```
- video_id: 配信URLに含まれているvideoID  

```
例
URL: https://www.youtube.com/watch?v=ABCDEFG&feature=youtu.be
video_id: ABCDEFG
```

- voting_time: 投票受付時間。単位は秒。デフォルトは30秒。  
投票はこの秒数以上で最も小さい5の倍数になります（コメントのポーリング感覚が5秒おきのため）。

- start_wait_time: 投票受付開始前の待機時間。単位は秒。デフォルトは0秒。

- multiple_balots: 一人一票ではなく、指定された数だけ複数投票可能になります。この複数投票は同じ選択肢を選ぶことができます。

- choice: 投票選択肢。複数指定可能  
投票対象を指定します。デフォルトでは半角数字の「1」と「2」  
ここで指定された値か、先頭から順番にふられる数字がコメントに含まれていると投票扱いになります。  
一人のユーザは一度まで投票できます。  
ここで指定された値が複数含まれている場合、一番初めにマッチした値に投票されます。

