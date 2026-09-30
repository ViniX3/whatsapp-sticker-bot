#!/usr/bin/env python3

import json
import shutil
import sys
from datetime import datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / "internal" / "rpg" / "data"
ITEMS = DATA / "items"
MATERIALS = DATA / "materials.json"

TARGETS = {
    "WORN": {"WEAPON": 34, "SHIELD": 33, "ARMOR": 33},
    "COMMON": {"WEAPON": 30, "SHIELD": 30, "ARMOR": 30},
    "RARE": {"WEAPON": 27, "SHIELD": 27, "ARMOR": 26},
    "EPIC": {"WEAPON": 17, "SHIELD": 17, "ARMOR": 16},
}

POWER = {
    "WORN": (35, 110),
    "COMMON": (120, 240),
    "RARE": (260, 460),
    "EPIC": (500, 800),
}

VALUE = {
    "WORN": (500, 3000),
    "COMMON": (3500, 12000),
    "RARE": (15000, 60000),
    "EPIC": (75000, 250000),
}

FORMS = {
    "WEAPON": [
        "Espada",
        "Machado",
        "Lança",
        "Adaga",
        "Maça",
        "Martelo",
        "Arco",
        "Foice",
        "Sabre",
        "Gládio",
        "Alabarda",
        "Clava",
    ],
    "SHIELD": [
        "Escudo",
        "Broquel",
        "Pavês",
        "Égide",
        "Escudo-Torre",
        "Escudo Redondo",
        "Baluarte",
        "Escudo de Guerra",
        "Escudo de Braço",
        "Escudo Reforçado",
    ],
    "ARMOR": [
        "Armadura",
        "Couraça",
        "Cota",
        "Brunea",
        "Peitoral",
        "Armadura Laminar",
        "Armadura de Placas",
        "Gibão de Guerra",
        "Vestes de Batalha",
        "Armadura Reforçada",
    ],
}

EPITHETS = {
    "WORN": [
        "do Recruta",
        "da Fronteira",
        "do Sobrevivente",
        "do Camponês",
        "do Viajante",
        "do Aprendiz",
    ],
    "COMMON": [
        "do Soldado",
        "do Ferreiro",
        "do Guarda",
        "do Mercenário",
        "do Caçador",
        "da Milícia",
    ],
    "RARE": [
        "do Veterano",
        "do Guardião",
        "da Montanha",
        "do Vale Antigo",
        "da Lua Cinzenta",
        "do Caçador Real",
    ],
    "EPIC": [
        "do Campeão",
        "da Tempestade",
        "do Caçador de Monstros",
        "da Forja Arcana",
        "do Herói Antigo",
        "da Guerra dos Reinos",
    ],
}

POOLS = {
    "WORN": {
        "WEAPON": [
            "stone",
            "wood_branch",
            "leather",
            "clay",
        ],
        "SHIELD": [
            "wood_branch",
            "stone",
            "leather",
            "clay",
        ],
        "ARMOR": [
            "leather",
            "wood_branch",
            "clay",
            "stone",
        ],
    },

    "COMMON": {
        "WEAPON": [
            "ingot_bronze",
            "ingot_iron",
            "wood_pine",
            "granite",
            "coal",
        ],
        "SHIELD": [
            "wood_pine",
            "ingot_bronze",
            "ingot_iron",
            "granite",
            "leather",
        ],
        "ARMOR": [
            "leather",
            "ingot_bronze",
            "ingot_iron",
            "wood_pine",
            "coal",
        ],
    },

    "RARE": {
        "WEAPON": [
            "ingot_steel",
            "ingot_silver",
            "ore_gold",
            "wolf_fang",
            "boar_tusk",
            "quartz",
        ],
        "SHIELD": [
            "wood_oak",
            "ingot_steel",
            "giant_turtle_shell",
            "hardened_leather",
            "ingot_silver",
            "quartz",
        ],
        "ARMOR": [
            "hardened_leather",
            "ingot_steel",
            "silk",
            "ingot_silver",
            "giant_turtle_shell",
            "quartz",
        ],
    },

    "EPIC": {
        "WEAPON": [
            "obsidian",
            "ore_mithril",
            "ice_wyvern_claw",
            "basilisk_eye",
            "hydra_venom",
            "dragon_scale",
            "demon_horn",
            "golem_core",
        ],
        "SHIELD": [
            "ore_mithril",
            "dragon_scale",
            "golem_core",
            "giant_turtle_shell",
            "obsidian",
            "griffin_feather",
            "sapphire",
        ],
        "ARMOR": [
            "arcane_fiber",
            "ore_mithril",
            "dragon_scale",
            "griffin_feather",
            "basilisk_eye",
            "ruby",
            "emerald",
        ],
    },
}

DISPLAY = {
    "wood_branch": "Madeira Rústica",
    "wood_pine": "Pinheiro",
    "wood_oak": "Carvalho",

    "ingot_bronze": "Bronze",
    "ingot_iron": "Ferro",
    "ingot_steel": "Aço",
    "ingot_silver": "Prata",

    "ore_gold": "Ouro",
    "ore_mithril": "Mithril",

    "ice_wyvern_claw": "Wyvern Gélido",
    "basilisk_eye": "Basilisco",
    "hydra_venom": "Hidra",
    "dragon_scale": "Escamas de Dragão",
    "demon_horn": "Chifre Demoníaco",
    "golem_core": "Núcleo de Golem",
    "giant_turtle_shell": "Casco de Tartaruga",
    "griffin_feather": "Penas de Grifo",

    "arcane_fiber": "Fibra Arcana",
}


def die(message):
    print(
        f"ERRO: {message}",
        file=sys.stderr,
    )

    sys.exit(1)


def load_json(path):
    try:
        return json.loads(
            path.read_text(
                encoding="utf-8",
            )
        )

    except FileNotFoundError:
        die(
            f"arquivo não encontrado: {path}"
        )

    except json.JSONDecodeError as exc:
        die(
            f"JSON inválido em {path}: {exc}"
        )


def write_json(
    path,
    data,
):
    tmp = path.with_name(
        path.name + ".tmp"
    )

    tmp.write_text(
        json.dumps(
            data,
            ensure_ascii=False,
            indent=2,
        )
        + "\n",
        encoding="utf-8",
    )

    tmp.replace(
        path,
    )


def interpolate(
    start,
    end,
    index,
    total,
):
    if total <= 1:
        return start

    return (
        start
        + (
            (end - start)
            * index
            // (total - 1)
        )
    )


def item_stats(
    item_type,
    power,
):
    ratio = {
        "WEAPON": 0.87,
        "SHIELD": 0.12,
        "ARMOR": 0.22,
    }[
        item_type
    ]

    attack = round(
        power * ratio
    )

    return (
        attack,
        power - attack,
    )


def material_name(
    material_id,
    materials,
):
    if material_id in DISPLAY:
        return DISPLAY[
            material_id
        ]

    name = materials[
        material_id
    ]["name"]

    for prefix in (
        "Minério de ",
        "Lingote de ",
        "Madeira de ",
        "Cristal de ",
    ):
        if name.startswith(
            prefix
        ):
            return name[
                len(prefix):
            ]

    return name


def make_recipe(
    pool,
    index,
    rarity,
):
    settings = {
        "WORN": (
            2,
            [2, 1],
        ),
        "COMMON": (
            3,
            [3, 2, 1],
        ),
        "RARE": (
            3,
            [4, 3, 2],
        ),
        "EPIC": (
            3,
            [5, 4, 2],
        ),
    }

    size, quantities = settings[
        rarity
    ]

    selected = []

    for offset in (
        0,
        1,
        3,
        5,
    ):
        material_id = pool[
            (index + offset)
            % len(pool)
        ]

        if material_id not in selected:
            selected.append(
                material_id
            )

        if len(selected) == size:
            break

    if len(selected) != size:
        die(
            f"pool insuficiente para {rarity}"
        )

    result = []

    for position, material_id in enumerate(
        selected
    ):
        quantity = quantities[
            position
        ]

        if position == 0:
            quantity += (
                index % 2
            )

        result.append(
            {
                "material_id":
                    material_id,

                "quantity":
                    quantity,
            }
        )

    return result


def generate_rarity(
    rarity,
    materials,
):
    result = []

    total = sum(
        TARGETS[
            rarity
        ].values()
    )

    cursor = 0

    for item_type in (
        "WEAPON",
        "SHIELD",
        "ARMOR",
    ):
        count = TARGETS[
            rarity
        ][
            item_type
        ]

        forms = FORMS[
            item_type
        ]

        epithets = EPITHETS[
            rarity
        ]

        pool = POOLS[
            rarity
        ][
            item_type
        ]

        for index in range(
            count
        ):
            power = interpolate(
                *POWER[rarity],
                cursor,
                total,
            )

            value = interpolate(
                *VALUE[rarity],
                cursor,
                total,
            )

            attack, defense = (
                item_stats(
                    item_type,
                    power,
                )
            )

            recipe = make_recipe(
                pool,
                index,
                rarity,
            )

            primary = recipe[
                0
            ][
                "material_id"
            ]

            theme = material_name(
                primary,
                materials,
            )

            form = forms[
                index
                % len(forms)
            ]

            epithet = epithets[
                (
                    index
                    // len(forms)
                )
                % len(epithets)
            ]

            monster_based = any(
                materials[
                    entry[
                        "material_id"
                    ]
                ][
                    "source"
                ]
                in (
                    "MONSTER_DROP",
                    "BOSS_DROP",
                )
                for entry
                in recipe
            )

            if monster_based:
                description = (
                    f"{form} criado com "
                    f"{theme} e troféus "
                    "obtidos de criaturas "
                    "perigosas."
                )

                origin = (
                    "Forja de Troféus"
                )

            else:
                description = (
                    f"{form} produzido com "
                    f"{theme} e recursos "
                    "extraídos das terras "
                    "do reino."
                )

                origin = (
                    "Forja do Reino"
                )

            result.append(
                {
                    "id":
                        f"{rarity.lower()}_"
                        f"{item_type.lower()}_"
                        f"{index + 1:03d}",

                    "name":
                        f"{form} de "
                        f"{theme} "
                        f"{epithet}",

                    "type":
                        item_type,

                    "rarity":
                        rarity,

                    "power":
                        power,

                    "attack":
                        attack,

                    "defense":
                        defense,

                    "description":
                        description,

                    "origin":
                        origin,

                    "value":
                        value,

                    "currency":
                        "GOLD",

                    "source":
                        "CRAFT",

                    "craft_materials":
                        recipe,
                }
            )

            cursor += 1

    if len(result) != total:
        die(
            f"{rarity}: esperado "
            f"{total}, gerado "
            f"{len(result)}"
        )

    return result


def validate(
    generated,
    materials,
):
    items = [
        item
        for group
        in generated.values()
        for item
        in group
    ]

    if len(items) != 320:
        die(
            "esperados 320 itens, "
            f"gerados {len(items)}"
        )

    ids = [
        item["id"]
        for item
        in items
    ]

    names = [
        item["name"]
        for item
        in items
    ]

    if len(ids) != len(
        set(ids)
    ):
        die(
            "existem IDs duplicados"
        )

    if len(names) != len(
        set(names)
    ):
        die(
            "existem nomes de "
            "equipamento duplicados"
        )

    for item in items:
        if item[
            "source"
        ] != "CRAFT":
            die(
                f"{item['id']} "
                "deveria ser CRAFT"
            )

        if (
            item["attack"]
            + item["defense"]
            != item["power"]
        ):
            die(
                "atributos inconsistentes "
                f"em {item['id']}"
            )

        seen = set()

        for requirement in item[
            "craft_materials"
        ]:
            material_id = (
                requirement[
                    "material_id"
                ]
            )

            if material_id not in materials:
                die(
                    f"{item['id']} usa "
                    "material inexistente: "
                    f"{material_id}"
                )

            if material_id in seen:
                die(
                    f"{item['id']} repete "
                    f"o material "
                    f"{material_id}"
                )

            if requirement[
                "quantity"
            ] <= 0:
                die(
                    f"{item['id']} possui "
                    "quantidade inválida"
                )

            seen.add(
                material_id
            )


def main():
    material_list = load_json(
        MATERIALS
    )

    materials = {
        item["id"]: item
        for item
        in material_list
    }

    if len(
        materials
    ) != len(
        material_list
    ):
        die(
            "materials.json possui "
            "IDs duplicados"
        )

    for rarity in TARGETS:
        for pool in POOLS[
            rarity
        ].values():
            for material_id in pool:
                if material_id not in materials:
                    die(
                        "material inexistente "
                        "no gerador: "
                        f"{material_id}"
                    )

    generated = {
        "worn.json":
            generate_rarity(
                "WORN",
                materials,
            ),

        "common.json":
            generate_rarity(
                "COMMON",
                materials,
            ),

        "rare.json":
            generate_rarity(
                "RARE",
                materials,
            ),

        "epic.json":
            generate_rarity(
                "EPIC",
                materials,
            ),
    }

    validate(
        generated,
        materials,
    )

    stamp = datetime.now().strftime(
        "%Y%m%d-%H%M%S"
    )

    ITEMS.mkdir(
        parents=True,
        exist_ok=True,
    )

    for filename, data in generated.items():
        path = ITEMS / filename

        if path.exists():
            shutil.copy2(
                path,
                path.with_name(
                    path.name
                    + ".before-rpg-base-generation-"
                    + stamp
                ),
            )

        write_json(
            path,
            data,
        )

    print(
        "Equipamentos base gerados "
        "com sucesso."
    )

    print(
        "WORN:    100"
    )

    print(
        "COMMON:   90"
    )

    print(
        "RARE:     80"
    )

    print(
        "EPIC:     50"
    )

    print(
        "TOTAL:   320"
    )


if __name__ == "__main__":
    main()
