#!/usr/bin/env python3

import argparse
import json
import re
import shutil
import sys
import tempfile
import unicodedata

from datetime import datetime
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]

QUIZ_DIR = (
    REPO_ROOT
    / "internal"
    / "quiz"
    / "questions"
)

DIFFICULTIES = {
    "super_easy": "super_easy.json",
    "easy": "easy.json",
    "medium": "medium.json",
    "hard": "hard.json",
    "super_hard": "super_hard.json",
    "insane": "insane.json",
}

ALIASES = {
    "super_easy": "super_easy",
    "super_facil": "super_easy",

    "easy": "easy",
    "facil": "easy",

    "medium": "medium",
    "medio": "medium",

    "hard": "hard",
    "dificil": "hard",

    "super_hard": "super_hard",
    "super_dificil": "super_hard",

    "insane": "insane",
    "insano": "insane",
}


def normalize_text(value):
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

    value = re.sub(
        r"\s+",
        " ",
        value,
    )

    return value


def normalize_key(value):
    value = normalize_text(value)

    value = re.sub(
        r"[^a-z0-9]+",
        "_",
        value,
    )

    return value.strip("_")


def resolve_difficulty(value):
    key = normalize_key(value)

    difficulty = ALIASES.get(key)

    if difficulty is None:
        valid = ", ".join(
            DIFFICULTIES.keys()
        )

        raise ValueError(
            f"Dificuldade inválida: {value}. "
            f"Valores aceitos: {valid}"
        )

    return difficulty


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


def validate_existing_database():
    all_questions = {}
    normalized_questions = {}

    for difficulty, filename in DIFFICULTIES.items():
        path = QUIZ_DIR / filename

        data = load_json(path)

        if not isinstance(data, list):
            raise ValueError(
                f"{filename} não contém uma lista."
            )

        for index, item in enumerate(
            data,
            start=1,
        ):
            validate_question(
                item,
                require_id=True,
                source=f"{filename}:{index}",
            )

            question_id = item["id"]

            if question_id in all_questions:
                raise ValueError(
                    "ID duplicado encontrado: "
                    f"{question_id}"
                )

            all_questions[question_id] = (
                difficulty,
                item,
            )

            normalized = normalize_text(
                item["question"]
            )

            if normalized in normalized_questions:
                previous = normalized_questions[
                    normalized
                ]

                raise ValueError(
                    "Pergunta duplicada no banco atual:\n"
                    f"  {previous}\n"
                    f"  {filename}:{index}\n"
                    f"  {item['question']}"
                )

            normalized_questions[
                normalized
            ] = f"{filename}:{index}"

    return (
        all_questions,
        normalized_questions,
    )


def validate_question(
    item,
    require_id=False,
    source="entrada",
):
    if not isinstance(item, dict):
        raise ValueError(
            f"{source}: pergunta precisa ser um objeto JSON."
        )

    if require_id:
        question_id = item.get("id")

        if (
            not isinstance(question_id, str)
            or not question_id.strip()
        ):
            raise ValueError(
                f"{source}: campo 'id' inválido."
            )

    category = item.get("category")

    if (
        not isinstance(category, str)
        or not category.strip()
    ):
        raise ValueError(
            f"{source}: campo 'category' é obrigatório."
        )

    question = item.get("question")

    if (
        not isinstance(question, str)
        or not question.strip()
    ):
        raise ValueError(
            f"{source}: campo 'question' é obrigatório."
        )

    options = item.get("options")

    if (
        not isinstance(options, list)
        or len(options) != 4
    ):
        raise ValueError(
            f"{source}: devem existir exatamente 4 alternativas."
        )

    normalized_options = []

    for option_index, option in enumerate(
        options
    ):
        if (
            not isinstance(option, str)
            or not option.strip()
        ):
            raise ValueError(
                f"{source}: alternativa "
                f"{option_index} está vazia."
            )

        normalized_options.append(
            normalize_text(option)
        )

    if len(set(normalized_options)) != 4:
        raise ValueError(
            f"{source}: existem alternativas duplicadas."
        )

    correct = item.get("correct")

    if isinstance(correct, str):
        value = correct.strip().upper()

        letter_map = {
            "A": 0,
            "B": 1,
            "C": 2,
            "D": 3,
        }

        if value in letter_map:
            correct = letter_map[value]

    if (
        not isinstance(correct, int)
        or isinstance(correct, bool)
        or correct < 0
        or correct > 3
    ):
        raise ValueError(
            f"{source}: 'correct' precisa ser "
            "0, 1, 2 ou 3 (também aceitamos A/B/C/D)."
        )

    return {
        "category": category.strip(),
        "question": question.strip(),
        "options": [
            option.strip()
            for option in options
        ],
        "correct": correct,
    }


def next_id_number(
    data,
    difficulty,
):
    pattern = re.compile(
        rf"^{re.escape(difficulty)}_(\d+)$"
    )

    highest = 0

    for item in data:
        match = pattern.match(
            item.get("id", "")
        )

        if not match:
            continue

        number = int(
            match.group(1)
        )

        highest = max(
            highest,
            number,
        )

    return highest + 1


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

    # Validação final antes de substituir.
    with temp_path.open(
        "r",
        encoding="utf-8",
    ) as file:
        json.load(file)

    temp_path.replace(path)


def main():
    parser = argparse.ArgumentParser(
        description=(
            "Adiciona perguntas ao banco do !quiz "
            "com validação e proteção contra duplicidades."
        )
    )

    parser.add_argument(
        "difficulty",
        help=(
            "Dificuldade: super_easy, easy, medium, "
            "hard, super_hard ou insane. "
            "Também aceita nomes em português."
        ),
    )

    parser.add_argument(
        "input",
        type=Path,
        help=(
            "Arquivo JSON contendo uma lista "
            "de novas perguntas."
        ),
    )

    args = parser.parse_args()

    try:
        difficulty = resolve_difficulty(
            args.difficulty
        )

        if not QUIZ_DIR.exists():
            raise ValueError(
                f"Diretório não encontrado: {QUIZ_DIR}"
            )

        input_path = args.input.resolve()

        new_questions = load_json(
            input_path
        )

        if not isinstance(
            new_questions,
            list,
        ):
            raise ValueError(
                "O arquivo de entrada precisa conter "
                "uma lista JSON."
            )

        if len(new_questions) == 0:
            raise ValueError(
                "O arquivo de entrada está vazio."
            )

        _, known_questions = (
            validate_existing_database()
        )

        target_path = (
            QUIZ_DIR
            / DIFFICULTIES[difficulty]
        )

        target_data = load_json(
            target_path
        )

        next_number = next_id_number(
            target_data,
            difficulty,
        )

        added = []
        skipped = []

        for position, raw_item in enumerate(
            new_questions,
            start=1,
        ):
            source = (
                f"{input_path.name}:{position}"
            )

            item = validate_question(
                raw_item,
                source=source,
            )

            normalized_question = (
                normalize_text(
                    item["question"]
                )
            )

            if normalized_question in known_questions:
                skipped.append(
                    (
                        item["question"],
                        "pergunta já existente",
                    )
                )

                continue

            question_id = (
                f"{difficulty}_"
                f"{next_number:03d}"
            )

            new_item = {
                "id": question_id,
                "category": item["category"],
                "question": item["question"],
                "options": item["options"],
                "correct": item["correct"],
            }

            target_data.append(
                new_item
            )

            known_questions[
                normalized_question
            ] = question_id

            added.append(
                new_item
            )

            next_number += 1

        if not added:
            print()
            print(
                "Nenhuma pergunta nova foi adicionada."
            )

            if skipped:
                print()
                print("Ignoradas:")

                for question, reason in skipped:
                    print(
                        f"  - {question} ({reason})"
                    )

            return

        timestamp = datetime.now().strftime(
            "%Y%m%d-%H%M%S"
        )

        backup = target_path.with_name(
            target_path.name
            + ".before-content-update-"
            + timestamp
        )

        shutil.copy2(
            target_path,
            backup,
        )

        atomic_write_json(
            target_path,
            target_data,
        )

        # Valida todo o banco novamente
        # após a escrita.
        validate_existing_database()

        print()
        print(
            "================================"
        )
        print(
            "Banco do Quiz atualizado"
        )
        print(
            "================================"
        )

        print(
            f"Dificuldade: {difficulty}"
        )

        print(
            f"Adicionadas: {len(added)}"
        )

        print(
            f"Ignoradas:   {len(skipped)}"
        )

        print(
            f"Total agora: {len(target_data)}"
        )

        print()
        print("IDs adicionados:")

        for item in added:
            print(
                f"  {item['id']} - "
                f"{item['question']}"
            )

        if skipped:
            print()
            print("Ignoradas:")

            for question, reason in skipped:
                print(
                    f"  - {question} ({reason})"
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
