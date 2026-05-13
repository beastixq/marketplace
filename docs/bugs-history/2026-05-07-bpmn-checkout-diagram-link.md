# BPMN Checkout Diagram Link Used Wrong Artifact Type

Status: fixed
Fixed: 2026-05-07
Commits: [`5160933`](https://github.com/beastixq/marketplace/commit/5160933), [`f3b45a1`](https://github.com/beastixq/marketplace/commit/f3b45a1)
Area: docs / diagrams

## Bug

README пытался встроить BPMN checkout diagram как PDF image:
`diagrams/out/BPMN_Checkout.pdf`. Markdown image embedding для PDF не
отображает диаграмму как обычную картинку.

## Cause

В README использовался путь к PDF artifact вместо generated PNG artifact.

## Fix

- Link заменён на `diagrams/out/BPMN_Checkout.png`.

## Verification

- Оба фикс-коммита меняют одну строку в `Readme.md`: `.pdf` -> `.png`.
