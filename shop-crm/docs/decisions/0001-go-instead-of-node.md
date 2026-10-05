# 0001. Go вместо Node.js/TypeScript в бэкенде

**Статус:** принято
**Дата:** 2026-10-05

## Контекст

ТЗ (разделы 2 и 8) предписывало Node.js + TypeScript + Express для бэкенда и React +
TypeScript для фронтенда. Пользователь отдельно указал: «Create repository and use golang
or kotlin».

## Решение

Бэкенд написан на Go: `chi` + `pgx` + `golang-migrate` + `go-playground/validator`.
Фронтенд остаётся на React + TypeScript + Vite + Tailwind + TanStack Query, как в ТЗ.

## Последствия

- Интеграционные тесты пишутся на стандартном `testing` с таблицами, без Vitest.
- Требование ТЗ «Vitest для юнит-тестов» не выполняется; вместо него `go test`.
- Требование «валидация через Zod» выполняется эквивалентно на уровне сервиса и
  `DisallowUnknownFields` на уровне HTTP.
- Kotlin не рассматривался: у Kotlin нет зрелого аналога `pgx` с `FOR UPDATE` в одном
  сервисе с comparably простой сборкой.

## Почему не Kotlin

Ktor + Exposed дают тот же результат, но требуют отдельного Gradle-проекта и JVM в
Docker-образе, что усложняет сборку без выигрыша для учебного проекта.