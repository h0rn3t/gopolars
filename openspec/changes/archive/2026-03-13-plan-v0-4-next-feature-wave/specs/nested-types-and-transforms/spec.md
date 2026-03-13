## ADDED Requirements

### Requirement: Nested DType contract SHALL cover list and struct workflows
Система SHALL формалізувати підтримку list/struct типів із явними правилами типізації, null поведінки та cast сумісності.

#### Scenario: Build dataframe with list and struct columns
- **WHEN** користувач створює dataframe з list/struct колонками
- **THEN** система коректно зберігає schema fidelity і доступність колонок для подальших операцій

### Requirement: Nested transforms SHALL support explode and flatten semantics
Система MUST підтримувати MVP nested трансформації (`explode`, `flatten` або еквівалентний контракт) для list/struct колонок.

#### Scenario: Explode list column into row-wise representation
- **WHEN** користувач викликає explode для list-колонки
- **THEN** система повертає очікувану кількість рядків і зберігає узгодженість інших колонок

### Requirement: Nested expression behavior SHALL be deterministic
Система SHALL забезпечувати детерміновану поведінку nested-виразів у eager/lazy режимах з еквівалентними результатами.

#### Scenario: Evaluate nested transform in lazy pipeline
- **WHEN** nested вираз додається у lazy pipeline і виконується collect
- **THEN** результат збігається з eager-еквівалентом для того самого вхідного датасету
