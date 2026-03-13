## Compatibility and Versioning Policy

### Change classes

- **Compatible**: Поведение и API совместимы, миграция не требуется.
- **Deprecating**: Старый API поддерживается в течение deprecation window с явным migration path.
- **Breaking**: Несовместимое изменение, требующее migration notes и release evidence.

### Deprecation windows

- Каждое deprecating изменение публикуется с целевым релизом удаления.
- Минимальное окно: один minor release между объявлением и удалением.
- Release notes должны содержать эквивалентный API путь или fallback стратегию.

### Breaking release gates

- Наличие migration документации обязательно.
- Наличие conformance evidence по затронутым capability обязательно.
- Наличие compatibility checklist в release artifacts обязательно.
