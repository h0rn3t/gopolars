"""Генерує POLARS_PARITY_TABLE.md — таблицю відповідності gopolars ↔ Python Polars.

Інвентаризація Polars береться інтроспекцією встановленого пакета, а не зашитим списком,
тому таблиця не може мовчки розійтися з реальним Polars. Go-сторона звіряється per-class:
DataFrame / LazyFrame / Series — з однойменних інтерфейсів у pkg/polars/types.go,
Expr — з методів із ресивером Expr у pkg/expr.

Запуск (потрібен інтерпретатор зі встановленим polars):

    .venv_polars_313/bin/python gen_parity_table.py
"""

import re
import sys
from pathlib import Path

try:
    import polars as pl
except ModuleNotFoundError:
    sys.exit(
        "polars не встановлено для цього інтерпретатора.\n"
        "Запусти генератор через venv з polars, напр.:\n"
        "    .venv_polars_313/bin/python gen_parity_table.py"
    )

# root відносно розташування цього скрипта — не залежить від машини.
root = Path(__file__).resolve().parent

# 1. Інвентаризація Python Polars — публічні атрибути класів.
PY_CLASSES = {
    "DataFrame": pl.DataFrame,
    "LazyFrame": pl.LazyFrame,
    "Series": pl.Series,
    "Expr": pl.Expr,
}
python_methods = {
    name: sorted(m for m in dir(cls) if not m.startswith("_"))
    for name, cls in PY_CLASSES.items()
}

# 2. Інвентаризація gopolars.
#
# Публічний контракт — це інтерфейси, а не реалізації: у реалізацій є допоміжні методи,
# яких немає в API. Expr — конкретний тип, тож для нього беремо методи з ресивером.
types_go = (root / "pkg/polars/types.go").read_text()


def interface_methods(name):
    """Методи, оголошені в `type <name> interface { ... }`.

    Зріз по балансу дужок, а не нежадібним регексом: тіло містить вкладені літерали
    типів, на першому ж `}` нежадібний варіант обірвався б посеред інтерфейсу.
    """
    header = re.search(rf"^type {name} interface \{{", types_go, re.M)
    if header is None:
        sys.exit(f"інтерфейс {name} не знайдено в pkg/polars/types.go")
    i, depth = header.end() - 1, 0
    while i < len(types_go):
        if types_go[i] == "{":
            depth += 1
        elif types_go[i] == "}":
            depth -= 1
            if depth == 0:
                break
        i += 1
    body = types_go[header.end():i]
    return {m.group(1) for m in re.finditer(r"^\s*([A-Z][A-Za-z0-9_]*)\(", body, re.M)}


def receiver_methods(glob, receiver):
    """Методи з ресивером `receiver` у файлах, що підпадають під glob (без _test.go)."""
    found = set()
    for path in sorted(root.glob(glob)):
        if path.name.endswith("_test.go"):
            continue
        for m in re.finditer(r"func \(([^)]*)\) ([A-Z][A-Za-z0-9_]*)\(", path.read_text()):
            if re.search(rf"\b{receiver}\b", m.group(1)):
                found.add(m.group(2))
    if not found:
        sys.exit(f"методи з ресивером {receiver} не знайдено ({glob})")
    return found


go_methods = {
    "DataFrame": interface_methods("DataFrame"),
    "LazyFrame": interface_methods("LazyFrame"),
    "Series": interface_methods("Series"),
    "Expr": receiver_methods("pkg/expr/*.go", "Expr"),
}


# 3. Зіставлення.
#
# Нормалізація знімає різницю регістру й підкреслень: sink_csv↔SinkCSV, truediv↔TrueDiv,
# drop_nans↔DropNaNs, qcut↔QCut. Без неї Go-акроніми читаються як прогалини.
def _norm(name):
    return name.replace("_", "").lower()


# Семантичні псевдоніми, де імена справді відрізняються, а не лише регістром.
# Впорядковані кортежі, а не множини: порядок кандидатів визначає, яке Go-ім'я потрапить
# у таблицю, а обхід множини рядків залежить від PYTHONHASHSEED — вивід став би недетермінованим.
ALIASES = {
    "dtype": ("datatype", "dtype"),
    "list": ("list", "arr"),
    "arr": ("arr", "list"),
}


def match_class(py_methods, go_names):
    """py-метод -> ім'я Go-методу, який його покриває (або None)."""
    by_norm = {_norm(g): g for g in sorted(go_names)}
    matched = {}
    for py in py_methods:
        for candidate in ALIASES.get(py, (_norm(py),)):
            if candidate in by_norm:
                matched[py] = by_norm[candidate]
                break
        else:
            matched[py] = None
    return matched


matches = {
    cls: match_class(python_methods[cls], go_methods[cls]) for cls in PY_CLASSES
}

# 4. Рендер.
OUT = []


def emit(line=""):
    OUT.append(line)


def print_table(name):
    matched = matches[name]
    emit(f"\n## {name}\n")
    emit("| Python (snake_case) | Go | Статус |")
    emit("|---|---|---|")
    missing = []
    for py in sorted(matched):
        go = matched[py]
        if go is None:
            missing.append(py)
            emit(f"| `{py}` | — | ❌ |")
        else:
            emit(f"| `{py}` | `{go}` | ✅ |")
    implemented = len(matched) - len(missing)
    pct = round(100 * implemented / len(matched), 1) if matched else 100.0
    emit(f"\n**Підсумок {name}:** {implemented}/{len(matched)} реалізовано (~{pct}%).")
    if missing:
        emit(f"\n**Не реалізовано ({len(missing)}):** " + ", ".join(f"`{m}`" for m in missing))
    emit()
    return implemented, len(matched)


emit(f"# Таблиця відповідності gopolars ↔ Python Polars {pl.__version__}")
emit()
emit("> Згенеровано `gen_parity_table.py`. Не редагувати вручну.")
emit(">")
emit("> Кожен клас звіряється лише з відповідним Go-типом: `DataFrame`/`LazyFrame`/`Series` —")
emit("> з інтерфейсів у `pkg/polars/types.go`, `Expr` — з методів із ресивером `Expr` у `pkg/expr`.")
emit("> Перевіряється наявність методу, не сигнатура й не семантика — останні покривають")
emit("> `test/parity` і `test/conformance`.")
emit()
emit("**Легенда:**")
emit("- ✅ Реалізовано — метод присутній у gopolars")
emit("- ❌ Не реалізовано — метод відсутній у gopolars")
emit("\n---")

totals = {cls: print_table(cls) for cls in ("DataFrame", "LazyFrame", "Series", "Expr")}

tot_impl = sum(i for i, _ in totals.values())
tot_all = sum(t for _, t in totals.values())

emit("## Загальний підсумок\n")
emit("| Клас | Реалізовано | Загалом | Відсоток |")
emit("|---|---|---|---|")
for cls, (impl, total) in totals.items():
    emit(f"| {cls} | {impl} | {total} | ~{round(100 * impl / total, 1)}% |")
emit(f"| **Разом** | **{tot_impl}** | **{tot_all}** | **~{round(100 * tot_impl / tot_all, 1)}%** |")
emit()

report = "\n".join(OUT)
out_path = root / "POLARS_PARITY_TABLE.md"
out_path.write_text(report)
print(f"[written] {out_path}")
print(f"polars={pl.__version__} implemented={tot_impl}/{tot_all}")
