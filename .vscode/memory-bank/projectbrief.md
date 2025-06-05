# Project Brief

## プロジェクト名
go-grpc-connect

## 概要
本プロジェクトは、Go言語によるバックエンドとReact（TypeScript）によるフロントエンド間で、gRPC通信（connect-go/connect-web）を用いたAPI連携を実現するサンプル/テンプレートプロジェクトです。  
Protocol BuffersによるAPI定義、Dockerによる開発環境の統一、モダンなフロントエンド開発環境（Vite, TypeScript, ESLint）を備えています。

## 目的
- GoバックエンドとReactフロントエンド間のgRPC通信の実装例を提供
- Protocol BuffersによるAPIスキーマ管理のベストプラクティスを示す
- Docker Composeによる開発・実行環境の簡易化
- フロントエンド/バックエンドの分離開発の推進

## スコープ
- Go（connect-go）によるgRPCサーバ実装
- React（connect-web）によるgRPCクライアント実装
- protoディレクトリでのAPIスキーマ管理
- Docker/Docker Composeによる開発環境構築
- サンプルAPI（greetサービス）の実装

## ゴール
- gRPC+connectプロトコルを用いたフルスタック開発の雛形を提供
- 新規プロジェクトのベースとして再利用可能な構成を確立
