## 1. API Parity Foundation

- [x] 1.1 Сформувати повний parity inventory по DataFrame/LazyFrame/Series API namespaces
- [x] 1.2 Додати missing API surface і вирівняти сигнатури з Polars Python
- [x] 1.3 Додати migration notes для потенційних BREAKING API alignment змін

## 2. Expression and DType Completion

- [x] 2.1 Реалізувати відсутні expression families і namespace-операції
- [x] 2.2 Розширити dtype систему nested/categorical/decimal/temporal типами
- [x] 2.3 Вирівняти null/NaN semantics у всіх операторах і агрегаціях

## 3. Analytics and Windowing

- [x] 3.1 Реалізувати window functions з partition/order semantics
- [x] 3.2 Додати rolling і dynamic temporal windows
- [x] 3.3 Реалізувати pivot/melt та advanced analytical aggregates

## 4. IO, Interoperability and Cloud

- [x] 4.1 Довести parity-read/write для CSV/JSON/Parquet/IPC
- [x] 4.2 Додати object-store backends для scan/read/write сценаріїв
- [x] 4.3 Розширити Arrow/interchange сумісність і roundtrip гарантії

## 5. Lazy Optimizer and Streaming

- [x] 5.1 Розширити optimizer rule set до parity baseline
- [x] 5.2 Реалізувати streaming execution paths з bounded-memory strategy
- [x] 5.3 Стабілізувати Explain контракт для logical/optimized/physical stages

## 6. Conformance, Benchmarks and Release Gates

- [x] 6.1 Побудувати differential conformance suite проти Python Polars
- [x] 6.2 Додати parity coverage matrix і автоматичні quality thresholds
- [x] 6.3 Інтегрувати perf/stability gates у CI для release candidate v0.2
