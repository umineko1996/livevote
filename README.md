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
usage: youtubelivevote -id video_id [-t voting_time] [-s start_wait_time] [choice1 choice2, ...]
```
- video_id: 配信URLに含まれているvideoID  

```
例
URL: https://www.youtube.com/watch?v=ABCDEFG&feature=youtu.be
video_id: ABCDEFG
```

- voting_time: 投票受付時間。単位は秒。デフォルトは30秒。  
投票はこの秒数以上で最も小さい5の倍数になります（コメントのポーリング感覚が5秒おきのため）。

- start_wait_time: 投票受付開始前の待機時間。単位は秒。デフォルトは3秒。

- choice: 投票選択肢。複数指定可能  
投票対象を指定します。デフォルトでは半角数字の「1」と「2」  
ここで指定された値か、先頭から順番にふられる数字がコメントに含まれていると投票扱いになります。  
一人のユーザは一度まで投票できます。  
ここで指定された値が複数含まれている場合、一番初めにマッチした値に投票されます。

```
例
youtubelivevote.exe -id ABCDEFG -t 30 あ い う
youtubelivevote.exe -id uHHdlb9qzZs -t 30 あ い う
start voting in 3s seconds... // コマンド実行から3秒後のコメントから集計を始まる
1."あ" // 引数に与えられた選択肢と対応する数字が表示される
2."い"
3."う"

vote start!
--------------------
time remaining 30 sec
delay 5 sec...
vote 1."あ"! ユーザ名(userID) text:"あいうえお" // コメント「あいうえお」の場合、一番初めの文字である「あ」にマッチする
time remaining 25 sec
vote 1."あ"! ユーザ名(userID) text:"1" // 「あ」に対して振られ番号「1」にマッチする
vote 2."い"! ユーザ名(userID) text:"2" // 一人一度の投票なのですべてのuserIDは一意になる
delay 5 sec...

（中略）// 5秒毎に同様の処理が実行される

time remaining 5 sec
delay 5 sec...
--------------------
vote end!

total vote 3
--------------------
1."あ" vote 2
2."い" vote 1
3."う" vote 0
--------------------
Winning vote 2
1."あ" // 一番投票数が多い選択肢が表示されます
```
