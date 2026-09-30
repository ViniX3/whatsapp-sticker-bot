#!/usr/bin/env python3

import json
import shutil
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent

WORN_FILE = (
    ROOT
    / "internal"
    / "rpg"
    / "data"
    / "items"
    / "worn.json"
)

BACKUP_FILE = Path(
    str(WORN_FILE) + ".before-starter-set"
)


STARTER_ITEMS = {
    "WEAPON": {
        "id": "starter_recruit_weapon",
        "name": "Espada do Recruta",
        "power": 50,
        "attack": 45,
        "defense": 5,
        "description": (
            "Uma espada simples entregue aos aventureiros "
            "que iniciam sua jornada."
        ),
        "origin": "Kit inicial do aventureiro",
    },

    "SHIELD": {
        "id": "starter_recruit_shield",
        "name": "Escudo do Recruta",
        "power": 45,
        "attack": 5,
        "defense": 40,
        "description": (
            "Um escudo simples usado pelos aventureiros "
            "em seus primeiros desafios."
        ),
        "origin": "Kit inicial do aventureiro",
    },

    "ARMOR": {
        "id": "starter_recruit_armor",
        "name": "Armadura do Recruta",
        "power": 60,
        "attack": 5,
        "defense": 55,
        "description": (
            "Uma armadura simples preparada para quem "
            "está começando sua jornada."
        ),
        "origin": "Kit inicial do aventureiro",
    },
}


def load_items():
    if not WORN_FILE.exists():
        print(
            f"ERRO: arquivo não encontrado: {WORN_FILE}"
        )
        sys.exit(1)

    with WORN_FILE.open(
        "r",
        encoding="utf-8",
    ) as file:
        data = json.load(file)

    if isinstance(data, list):
        return data, data

    if (
        isinstance(data, dict)
        and isinstance(
            data.get("items"),
            list,
        )
    ):
        return data, data["items"]

    print(
        "ERRO: formato de worn.json não reconhecido."
    )
    sys.exit(1)


def find_item(
    items,
    item_type,
    starter_id,
):
    # Se o script já foi executado anteriormente,
    # localiza diretamente pelo ID oficial.
    for item in items:
        if item.get("id") == starter_id:
            return item

    # Caso contrário, reserva o primeiro equipamento
    # WORN disponível daquele tipo.
    for item in items:
        if (
            str(
                item.get(
                    "type",
                    "",
                )
            ).upper()
            == item_type
        ):
            return item

    return None


def validate_unique_ids(
    items,
):
    seen = set()

    for item in items:
        item_id = item.get("id")

        if not item_id:
            print(
                "ERRO: equipamento sem ID encontrado."
            )
            sys.exit(1)

        if item_id in seen:
            print(
                f"ERRO: ID duplicado: {item_id}"
            )
            sys.exit(1)

        seen.add(item_id)


def main():
    root_data, items = load_items()

    if len(items) != 100:
        print(
            "ERRO: worn.json deveria possuir "
            f"100 itens, mas possui {len(items)}."
        )
        sys.exit(1)

    selected = {}

    for item_type, starter in STARTER_ITEMS.items():
        item = find_item(
            items,
            item_type,
            starter["id"],
        )

        if item is None:
            print(
                "ERRO: não foi encontrado equipamento "
                f"WORN do tipo {item_type}."
            )
            sys.exit(1)

        selected[item_type] = item

    # Evita selecionar acidentalmente o mesmo objeto.
    selected_objects = {
        id(item)
        for item in selected.values()
    }

    if len(selected_objects) != 3:
        print(
            "ERRO: os três equipamentos iniciais "
            "não são distintos."
        )
        sys.exit(1)

    print(
        "Equipamentos que serão reservados:"
    )

    for item_type, item in selected.items():
        print(
            f" - {item_type}: "
            f"{item.get('id')} -> "
            f"{STARTER_ITEMS[item_type]['id']}"
        )

    shutil.copy2(
        WORN_FILE,
        BACKUP_FILE,
    )

    for item_type, starter in STARTER_ITEMS.items():
        item = selected[item_type]

        # Preservamos source e craft_materials existentes
        # para manter compatibilidade com o catálogo atual.
        item["id"] = starter["id"]
        item["name"] = starter["name"]
        item["type"] = item_type
        item["rarity"] = "WORN"

        item["power"] = starter["power"]
        item["attack"] = starter["attack"]
        item["defense"] = starter["defense"]

        item["description"] = starter[
            "description"
        ]

        item["origin"] = starter[
            "origin"
        ]

        # O jogador deve possuir no máximo
        # uma unidade do equipamento inicial.
        if "unique" in item:
            item["unique"] = True

    validate_unique_ids(
        items,
    )

    for item_type, starter in STARTER_ITEMS.items():
        item = selected[item_type]

        expected_power = (
            item.get("attack", 0)
            + item.get("defense", 0)
        )

        if (
            item.get("power")
            != expected_power
        ):
            print(
                "ERRO: ATQ + DEF não corresponde "
                f"ao PC de {starter['id']}."
            )
            sys.exit(1)

    with WORN_FILE.open(
        "w",
        encoding="utf-8",
    ) as file:
        json.dump(
            root_data,
            file,
            ensure_ascii=False,
            indent=2,
        )

        file.write(
            "\n"
        )

    print()
    print(
        "Set do Recruta aplicado com sucesso."
    )

    print(
        f"Backup: {BACKUP_FILE}"
    )

    print()
    print(
        "Equipamentos oficiais:"
    )

    for starter in STARTER_ITEMS.values():
        print(
            f" - {starter['id']}: "
            f"{starter['name']} "
            f"({starter['power']} PC)"
        )

    print()
    print(
        f"Total WORN preservado: {len(items)}"
    )


if __name__ == "__main__":
    main()
