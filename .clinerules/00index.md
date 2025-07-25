# ファイルの役割と重複回避

## projectbrief.md

- プロジェクトの基本的な要件と目標を定義する
- 他のすべてのファイルの基盤となる
- プロジェクトの範囲と成功基準を明確に示す
- memory-bank/ 内の他のファイルはこのファイルを参照する

## 01Overview.md

- プロジェクト全体の技術的な概要を提供する
- プロジェクトの構造、コードスタイル、開発ワークフローなどの技術的な情報を含む
- memory-bank/ 内のファイルと重複する情報がある場合は、詳細情報を memory-bank/ に移動し、01Overview.md からは参照するようにする
- 特に productContext.md と重複する「プロジェクト概要」や「主要機能」などの情報は、memory-bank/ を参照するようにする

## MemoryBank.md

- Cline の動作方法と記憶管理に関する指示を提供する
- memory-bank/ ファイルの目的と管理方法を説明する
- プロジェクトの具体的な技術情報は含めず、Cline の記憶管理に焦点を当てる

## 重複や矛盾がある場合

- 重複がある場合は、01Overview.md から該当情報を削除し、memory-bank/ への参照に置き換える
- 矛盾がある場合は、ユーザーに確認して解決する
