#!/usr/bin/env python3

import argparse
import json
import shutil
import sys
import tempfile
import unicodedata

from datetime import datetime
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]

WORDS_PATH = (
    REPO_ROOT
    / "internal"
    / "forca"
    / "words.json"
)


def normalize(value):
    value = str(value).strip().lower()

    value = unicodedata.normalize(
        "NFD",
        value,
    )

    value = "".join(
        char
        for char in value
        if unicodedata.category(char) != "Mn"
    )

    return " ".join(
        value.split()
    )


def load_json(path):
    try:
        with path.open(
            "r",
            encoding="utf-8",
        ) as file:
            return json.load(file)

    except FileNotFoundError:
        raise ValueError(
            f"Arquivo não encontrado: {path}"
        )

    except json.JSONDecodeError as exc:
        raise ValueError(
            f"JSON inválido em {path}: {exc}"
        )


def validate_word(
    item,
    source,
):
    if not isinstance(
        item,
        dict,
    ):
        raise ValueError(
            f"{source}: registro precisa ser um objeto JSON."
        )

    word = item.get(
        "word",
        "",
    )

    category = item.get(
        "category",
        "",
    )

    if (
        not isinstance(word, str)
        or not word.strip()
    ):
        raise ValueError(
            f"{source}: campo 'word' inválido."
        )

    if (
        not isinstance(category, str)
        or not category.strip()
    ):
        raise ValueError(
            f"{source}: campo 'category' inválido."
        )

    return {
        "word": " ".join(
            word.strip().split()
        ),
        "category": category.strip(),
    }


def atomic_write_json(
    path,
    data,
):
    with tempfile.NamedTemporaryFile(
        mode="w",
        encoding="utf-8",
        dir=path.parent,
        prefix=path.name + ".tmp.",
        delete=False,
    ) as temporary:

        json.dump(
            data,
            temporary,
            ensure_ascii=False,
            indent=2,
        )

        temporary.write("\n")
        temporary.flush()

        temp_path = Path(
            temporary.name
        )

    # Validação antes de substituir o arquivo real.
    with temp_path.open(
        "r",
        encoding="utf-8",
    ) as file:
        json.load(file)

    temp_path.replace(
        path
    )


def main():
    parser = argparse.ArgumentParser(
        description=(
            "Adiciona novas palavras ao banco do !forca "
            "com validação e proteção contra duplicidades."
        )
    )

    parser.add_argument(
        "input",
        type=Path,
        help=(
            "Arquivo JSON contendo uma lista "
            "de novas palavras."
        ),
    )

    args = parser.parse_args()

    try:
        if not WORDS_PATH.exists():
            raise ValueError(
                f"Banco da Forca não encontrado: {WORDS_PATH}"
            )

        input_path = (
            args.input.resolve()
        )

        existing_data = load_json(
            WORDS_PATH
        )

        new_data = load_json(
            input_path
        )

        if not isinstance(
            existing_data,
            list,
        ):
            raise ValueError(
                "O words.json atual não contém uma lista."
            )

        if not isinstance(
            new_data,
            list,
        ):
            raise ValueError(
                "O arquivo de entrada precisa conter uma lista JSON."
            )

        if len(new_data) == 0:
            raise ValueError(
                "O arquivo de entrada está vazio."
            )

        existing_words = {}

        for position, raw_item in enumerate(
            existing_data,
            start=1,
        ):
            item = validate_word(
                raw_item,
                f"words.json:{position}",
            )

            key = normalize(
                item["word"]
            )

            if key in existing_words:
                raise ValueError(
                    "Palavra duplicada encontrada no banco atual: "
                    f"{item['word']}"
                )

            existing_words[
                key
            ] = item["word"]

        added = []
        skipped = []

        for position, raw_item in enumerate(
            new_data,
            start=1,
        ):
            item = validate_word(
                raw_item,
                f"{input_path.name}:{position}",
            )

            key = normalize(
                item["word"]
            )

            if key in existing_words:
                skipped.append(
                    item["word"]
                )

                continue

            existing_data.append(
                item
            )

            existing_words[
                key
            ] = item["word"]

            added.append(
                item
            )

        if not added:
            print()
            print(
                "Nenhuma palavra nova foi adicionada."
            )

            if skipped:
                print()
                print("Duplicadas:")

                for word in skipped:
                    print(
                        f"  - {word}"
                    )

            return

        timestamp = (
            datetime.now().strftime(
                "%Y%m%d-%H%M%S"
            )
        )

        backup = WORDS_PATH.with_name(
            WORDS_PATH.name
            + ".before-content-update-"
            + timestamp
        )

        shutil.copy2(
            WORDS_PATH,
            backup,
        )

        atomic_write_json(
            WORDS_PATH,
            existing_data,
        )

        # Validação final.
        final_data = load_json(
            WORDS_PATH
        )

        if len(final_data) != len(
            existing_data
        ):
            raise ValueError(
                "Falha na validação final do banco."
            )

        print()
        print(
            "================================"
        )
        print(
            "Banco da Forca atualizado"
        )
        print(
            "================================"
        )

        print(
            f"Adicionadas: {len(added)}"
        )

        print(
            f"Ignoradas:   {len(skipped)}"
        )

        print(
            f"Total agora: {len(final_data)}"
        )

        print()
        print("Palavras adicionadas:")

        for item in added:
            print(
                f"  - {item['word']} "
                f"[{item['category']}]"
            )

        if skipped:
            print()
            print("Duplicadas ignoradas:")

            for word in skipped:
                print(
                    f"  - {word}"
                )

        print()
        print(
            f"Backup: {backup}"
        )

    except ValueError as exc:
        print(
            f"ERRO: {exc}",
            file=sys.stderr,
        )

        sys.exit(1)


if __name__ == "__main__":
    main()
