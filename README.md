# cli-with-web-gui-sample

CLIツールにWebベースのGUI機能を追加するサンプル実装です。
JSON -> YAML 変換機能を持ったCLIツールを例に、引数なしで起動した場合にWeb GUIモードに切り替わる仕組みを紹介します。

## 機能

- JSON → YAML 変換
- CLIモード（従来のコマンドライン操作）
- Webモード（ブラウザベースのGUI）

## 使い方

### ビルド

```bash
go mod tidy
go build -o json2yaml .
```

### CLIモード

```bash
# 標準出力に結果を表示
./json2yaml sample.json

# ファイルに出力
./json2yaml sample.json output.yaml
```

### Webモード

```bash
# 引数なしで起動するとWebモード
./json2yaml

# または明示的にwebサブコマンド
./json2yaml web

# ポート指定
./json2yaml web --port 3000
```

Webモードでは:
1. ブラウザが自動で起動
2. ドラッグ&ドロップでファイルをアップロード可能
3. テキストエリアに直接JSONを貼り付け可能
4. 変換結果をダウンロード可能
5. ブラウザを閉じるとサーバが自動終了

## 技術的なポイント

- `embed`パッケージでHTML/CSS/JSをバイナリに埋め込み
- ハートビート監視でブラウザ終了を検知
- クロスプラットフォーム対応のブラウザ自動起動
