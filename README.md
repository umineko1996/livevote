# youtubelivevote
Youtubeライブストリーミングのチャットで投票を行うためのツールです。

# 使い方
初回実行時はYoutubeAPIを実行するためにGoogle OAuth2ログインを求められます。

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
start voting in 3s seconds...
if you stop vote halfway, please press enter key // 投票途中で締め切りたい場合、エンターキーを入力すれば次のポーリングタイミングで締め切る
1."あ" // 引数に与えられた選択肢と対応する数字が表示される
2."い"
3."う"

vote start!
--------------------
time remaining 30 sec
delay 5 sec... // 5秒待機後チャットを取得し、その間のチャットのうち投票にカウントされたチャットを表示する
vote 1."あ" ユーザ名(userID) text:"おえういあ" // 一番初めにチェックされる「あ」にマッチする
time remaining 25 sec
vote 1."あ" ユーザ名(userID) text:"1" // 「あ」に対して振られ番号「1」にマッチする
vote 2."い" ユーザ名(userID) text:"2" // 一人一度の投票なのですべてのuserIDは一意になる
delay 5 sec...

（中略）// 5秒毎に同様の処理が実行される

time remaining 5 sec
delay 5 sec...
--------------------
vote end!

total vote 3 // 選択肢全体への投票数
--------------------
1."あ" vote 2 // それぞれの選択肢への投票数
2."い" vote 1
3."う" vote 0
--------------------
Winning vote 2
1."あ" // 一番投票数が多い選択肢が表示されます
```
