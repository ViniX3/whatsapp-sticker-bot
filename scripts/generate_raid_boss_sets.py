#!/usr/bin/env python3

import json
import os
import shutil
from datetime import datetime
from pathlib import Path

ROOT = Path("internal/rpg/data")

LEGENDARY_FILE = ROOT / "items" / "legendary.json"
MYTHIC_FILE = ROOT / "items" / "mythic.json"
SETS_FILE = ROOT / "sets.json"
BOSSES_FILE = ROOT / "raid_bosses.json"


def effect(effect_type, value):
    descriptions = {
        "COMBAT_POWER_PERCENT":
            f"+{value}% de Poder de Combate.",

        "DROP_CHANCE_PERCENT":
            f"+{value}% de chance de drops.",

        "RARE_DROP_CHANCE_PERCENT":
            f"+{value}% de chance de drops raros.",

        "GAME_LUCK_PERCENT":
            f"+{value}% de Sorte.",

        "CRYSTAL_REWARD_PERCENT":
            f"+{value}% de Cristais Mágicos obtidos.",

        "STELLAR_STONE_CHANCE_PERCENT":
            f"+{value}% de chance de obter Pedras Estelares.",

        "DUNGEON_SUCCESS_PERCENT":
            f"+{value}% de chance de sucesso em Dungeons.",

        "BOSS_DAMAGE_PERCENT":
            f"+{value}% de dano contra Bosses.",

        "BLESSING_CHANCE_PERCENT":
            f"+{value}% de chance de receber uma bênção.",

        "DROP_RARITY_UPGRADE_CHANCE_PERCENT":
            f"+{value}% de chance de elevar a raridade dos drops.",
    }

    return {
        "type": effect_type,
        "value": value,
        "description": descriptions[effect_type],
    }


def bonus(required_pieces, *effects):
    return {
        "required_pieces": required_pieces,
        "effects": [
            effect(effect_type, value)
            for effect_type, value in effects
        ],
    }


SPECS = [
    # =====================================================
    # LEGENDARY
    # =====================================================

    {
        "boss_id": "obsidian_basilisk",
        "set_id": "legendary_obsidian_basilisk",
        "rarity": "LEGENDARY",
        "set_name": "Regalia do Basilisco de Obsidiana",
        "theme":
            "Relíquias negras impregnadas pelo veneno e pelo olhar petrificante do Basilisco de Obsidiana.",

        "weapon_name": "Lâmina do Olhar Petrificante",
        "shield_name": "Escudo de Obsidiana Venenosa",
        "armor_name": "Couraça do Basilisco Negro",

        "weapon": (1290, 1120, 170),
        "shield": (1170, 140, 1030),
        "armor": (1240, 270, 970),

        "bonuses": [
            bonus(
                2,
                ("RARE_DROP_CHANCE_PERCENT", 5),
            ),
            bonus(
                3,
                ("BOSS_DAMAGE_PERCENT", 8),
                ("DROP_CHANCE_PERCENT", 5),
            ),
        ],
    },

    {
        "boss_id": "abyssal_kraken",
        "set_id": "legendary_abyssal_kraken",
        "rarity": "LEGENDARY",
        "set_name": "Tesouros do Kraken Abissal",
        "theme":
            "Equipamentos moldados nas profundezas oceânicas onde o Kraken Abissal domina sem oposição.",

        "weapon_name": "Tridente das Profundezas",
        "shield_name": "Escudo dos Tentáculos Abissais",
        "armor_name": "Armadura da Fossa Oceânica",

        "weapon": (1300, 1130, 170),
        "shield": (1180, 150, 1030),
        "armor": (1250, 280, 970),

        "bonuses": [
            bonus(
                2,
                ("CRYSTAL_REWARD_PERCENT", 10),
            ),
            bonus(
                3,
                ("DUNGEON_SUCCESS_PERCENT", 7),
                ("DROP_CHANCE_PERCENT", 5),
            ),
        ],
    },

    {
        "boss_id": "crimson_chimera",
        "set_id": "legendary_crimson_chimera",
        "rarity": "LEGENDARY",
        "set_name": "Arsenal da Quimera Carmesim",
        "theme":
            "Um conjunto instável formado pela essência fundida das feras que compõem a Quimera Carmesim.",

        "weapon_name": "Garras da Quimera Carmesim",
        "shield_name": "Escudo das Três Feras",
        "armor_name": "Couraça da Fusão Carmesim",

        "weapon": (1310, 1140, 170),
        "shield": (1180, 160, 1020),
        "armor": (1260, 300, 960),

        "bonuses": [
            bonus(
                2,
                ("BOSS_DAMAGE_PERCENT", 8),
            ),
            bonus(
                3,
                ("COMBAT_POWER_PERCENT", 6),
                ("GAME_LUCK_PERCENT", 2),
            ),
        ],
    },

    {
        "boss_id": "ash_cerberus",
        "set_id": "legendary_ash_cerberus",
        "rarity": "LEGENDARY",
        "set_name": "Relíquias do Cérbero das Cinzas",
        "theme":
            "Armas e armaduras queimadas pelo fogo eterno do guardião de três cabeças.",

        "weapon_name": "Machado das Cinzas Tríplices",
        "shield_name": "Escudo do Portão Cinzento",
        "armor_name": "Armadura do Guardião Infernal",

        "weapon": (1320, 1150, 170),
        "shield": (1190, 150, 1040),
        "armor": (1260, 290, 970),

        "bonuses": [
            bonus(
                2,
                ("COMBAT_POWER_PERCENT", 7),
            ),
            bonus(
                3,
                ("BLESSING_CHANCE_PERCENT", 2),
                ("DROP_CHANCE_PERCENT", 6),
            ),
        ],
    },

    {
        "boss_id": "crystal_golem",
        "set_id": "legendary_crystal_golem",
        "rarity": "LEGENDARY",
        "set_name": "Núcleo do Golem de Cristal",
        "theme":
            "Equipamentos prismáticos formados pelos fragmentos do núcleo vivo do Golem de Cristal.",

        "weapon_name": "Martelo do Núcleo Prismático",
        "shield_name": "Bastião de Cristal",
        "armor_name": "Carapaça Prismática",

        "weapon": (1310, 1110, 200),
        "shield": (1210, 130, 1080),
        "armor": (1280, 230, 1050),

        "bonuses": [
            bonus(
                2,
                ("CRYSTAL_REWARD_PERCENT", 10),
            ),
            bonus(
                3,
                ("RARE_DROP_CHANCE_PERCENT", 6),
                ("DROP_RARITY_UPGRADE_CHANCE_PERCENT", 3),
            ),
        ],
    },

    {
        "boss_id": "onyx_spider_queen",
        "set_id": "legendary_onyx_spider_queen",
        "rarity": "LEGENDARY",
        "set_name": "Vestes da Rainha Aracnídea de Ônix",
        "theme":
            "Um conjunto sombrio tecido com fios minerais produzidos pela Rainha Aracnídea de Ônix.",

        "weapon_name": "Lâminas da Teia de Ônix",
        "shield_name": "Escudo da Viúva de Ônix",
        "armor_name": "Manto da Rainha Aracnídea",

        "weapon": (1330, 1160, 170),
        "shield": (1190, 170, 1020),
        "armor": (1270, 310, 960),

        "bonuses": [
            bonus(
                2,
                ("DROP_CHANCE_PERCENT", 7),
            ),
            bonus(
                3,
                ("GAME_LUCK_PERCENT", 2),
                ("RARE_DROP_CHANCE_PERCENT", 5),
            ),
        ],
    },

    {
        "boss_id": "eternal_labyrinth_minotaur",
        "set_id": "legendary_eternal_labyrinth_minotaur",
        "rarity": "LEGENDARY",
        "set_name": "Arsenal do Labirinto Eterno",
        "theme":
            "Equipamentos carregados pelo guardião que percorreu o Labirinto Eterno durante incontáveis eras.",

        "weapon_name": "Machado do Labirinto Eterno",
        "shield_name": "Escudo dos Corredores Infinitos",
        "armor_name": "Armadura do Guardião do Labirinto",

        "weapon": (1340, 1160, 180),
        "shield": (1210, 150, 1060),
        "armor": (1290, 270, 1020),

        "bonuses": [
            bonus(
                2,
                ("DUNGEON_SUCCESS_PERCENT", 7),
            ),
            bonus(
                3,
                ("COMBAT_POWER_PERCENT", 6),
                ("BOSS_DAMAGE_PERCENT", 7),
            ),
        ],
    },

    {
        "boss_id": "jade_dragon",
        "set_id": "legendary_jade_dragon",
        "rarity": "LEGENDARY",
        "set_name": "Regalia do Dragão de Jade",
        "theme":
            "Relíquias esculpidas em jade vivo e fortalecidas pelo sopro ancestral do Dragão de Jade.",

        "weapon_name": "Lâmina do Sopro de Jade",
        "shield_name": "Escudo das Escamas de Jade",
        "armor_name": "Regalia do Dragão de Jade",

        "weapon": (1360, 1180, 180),
        "shield": (1220, 170, 1050),
        "armor": (1310, 320, 990),

        "bonuses": [
            bonus(
                2,
                ("BOSS_DAMAGE_PERCENT", 8),
            ),
            bonus(
                3,
                ("DROP_RARITY_UPGRADE_CHANCE_PERCENT", 3),
                ("CRYSTAL_REWARD_PERCENT", 10),
            ),
        ],
    },

    # =====================================================
    # MYTHIC
    # =====================================================

    {
        "boss_id": "chronal_dragon",
        "set_id": "mythic_chronal_dragon",
        "rarity": "MYTHIC",
        "set_name": "Regalia do Dragão Cronal",
        "theme":
            "Equipamentos que existem simultaneamente em diversos instantes do tempo.",

        "weapon_name": "Espada da Última Hora",
        "shield_name": "Escudo do Instante Imóvel",
        "armor_name": "Armadura das Eras",

        "weapon": (1920, 1700, 220),
        "shield": (1720, 190, 1530),
        "armor": (1830, 410, 1420),

        "bonuses": [
            bonus(
                2,
                ("COMBAT_POWER_PERCENT", 10),
            ),
            bonus(
                3,
                ("DROP_RARITY_UPGRADE_CHANCE_PERCENT", 5),
                ("BOSS_DAMAGE_PERCENT", 10),
            ),
        ],
    },

    {
        "boss_id": "spectral_emperor",
        "set_id": "mythic_spectral_emperor",
        "rarity": "MYTHIC",
        "set_name": "Tesouros do Imperador Espectral",
        "theme":
            "Relíquias imperiais envoltas pelas almas das legiões que ainda obedecem ao Imperador Espectral.",

        "weapon_name": "Cetro das Legiões Espectrais",
        "shield_name": "Escudo do Trono Fantasma",
        "armor_name": "Vestes do Imperador Espectral",

        "weapon": (1940, 1710, 230),
        "shield": (1730, 190, 1540),
        "armor": (1840, 420, 1420),

        "bonuses": [
            bonus(
                2,
                ("BLESSING_CHANCE_PERCENT", 4),
            ),
            bonus(
                3,
                ("RARE_DROP_CHANCE_PERCENT", 10),
                ("CRYSTAL_REWARD_PERCENT", 15),
            ),
        ],
    },

    {
        "boss_id": "solar_archon",
        "set_id": "mythic_solar_archon",
        "rarity": "MYTHIC",
        "set_name": "Regalia do Arconte Solar",
        "theme":
            "Artefatos radiantes moldados a partir da energia concentrada de uma estrela.",

        "weapon_name": "Lança do Sol Absoluto",
        "shield_name": "Égide Solar",
        "armor_name": "Armadura do Arconte Radiante",

        "weapon": (1970, 1730, 240),
        "shield": (1750, 200, 1550),
        "armor": (1870, 430, 1440),

        "bonuses": [
            bonus(
                2,
                ("BOSS_DAMAGE_PERCENT", 12),
            ),
            bonus(
                3,
                ("CRYSTAL_REWARD_PERCENT", 18),
                ("DROP_CHANCE_PERCENT", 8),
            ),
        ],
    },

    {
        "boss_id": "end_herald",
        "set_id": "mythic_end_herald",
        "rarity": "MYTHIC",
        "set_name": "Vestígios do Arauto do Fim",
        "theme":
            "Equipamentos que carregam o presságio de um mundo prestes a desaparecer.",

        "weapon_name": "Foice do Último Presságio",
        "shield_name": "Escudo do Crepúsculo Final",
        "armor_name": "Armadura do Arauto do Fim",

        "weapon": (2000, 1760, 240),
        "shield": (1770, 200, 1570),
        "armor": (1900, 440, 1460),

        "bonuses": [
            bonus(
                2,
                ("STELLAR_STONE_CHANCE_PERCENT", 8),
            ),
            bonus(
                3,
                ("BOSS_DAMAGE_PERCENT", 10),
                ("DROP_CHANCE_PERCENT", 10),
            ),
        ],
    },

    {
        "boss_id": "infinity_guardian",
        "set_id": "mythic_infinity_guardian",
        "rarity": "MYTHIC",
        "set_name": "Arsenal do Guardião do Infinito",
        "theme":
            "Artefatos impossíveis cujas superfícies parecem se estender muito além de seus próprios limites.",

        "weapon_name": "Lâmina do Horizonte Infinito",
        "shield_name": "Escudo do Infinito",
        "armor_name": "Armadura do Guardião Eterno",

        "weapon": (2030, 1780, 250),
        "shield": (1800, 210, 1590),
        "armor": (1930, 450, 1480),

        "bonuses": [
            bonus(
                2,
                ("COMBAT_POWER_PERCENT", 12),
            ),
            bonus(
                3,
                ("STELLAR_STONE_CHANCE_PERCENT", 8),
                ("DROP_RARITY_UPGRADE_CHANCE_PERCENT", 5),
            ),
        ],
    },

    {
        "boss_id": "aurora_colossus",
        "set_id": "mythic_aurora_colossus",
        "rarity": "MYTHIC",
        "set_name": "Arsenal do Colosso da Aurora",
        "theme":
            "Equipamentos gigantescos cobertos por luzes que se movem como uma aurora no céu.",

        "weapon_name": "Martelo da Aurora Boreal",
        "shield_name": "Bastião da Aurora",
        "armor_name": "Armadura do Colosso Luminar",

        "weapon": (2060, 1810, 250),
        "shield": (1820, 220, 1600),
        "armor": (1960, 470, 1490),

        "bonuses": [
            bonus(
                2,
                ("CRYSTAL_REWARD_PERCENT", 18),
            ),
            bonus(
                3,
                ("BLESSING_CHANCE_PERCENT", 5),
                ("DROP_CHANCE_PERCENT", 10),
            ),
        ],
    },
]


def read_json(path):
    return json.loads(
        path.read_text(
            encoding="utf-8",
        )
    )


def write_json_atomic(path, data):
    temp = Path(
        str(path) + ".tmp"
    )

    temp.write_text(
        json.dumps(
            data,
            ensure_ascii=False,
            indent=2,
        ) + "\n",
        encoding="utf-8",
    )

    os.replace(
        temp,
        path,
    )


def create_item(
    spec,
    suffix,
    item_type,
    item_name,
    stats,
):
    power, attack, defense = stats

    if power != attack + defense:
        raise RuntimeError(
            f"{spec['set_id']}_{suffix}: "
            f"power {power} != "
            f"attack {attack} + defense {defense}"
        )

    boss_name = spec["_boss_name"]

    descriptions = {
        "weapon":
            f"Arma obtida ao derrotar {boss_name}, carregada com parte de seu poder.",

        "shield":
            f"Escudo formado a partir da essência de {boss_name}.",

        "armor":
            f"Armadura impregnada pelo poder de {boss_name}.",
    }

    return {
        "id":
            f"{spec['set_id']}_{suffix}",

        "name":
            item_name,

        "type":
            item_type,

        "rarity":
            spec["rarity"],

        "power":
            power,

        "attack":
            attack,

        "defense":
            defense,

        "description":
            descriptions[suffix],

        "origin":
            f"Raid Boss: {boss_name}",

        "value":
            0,

        "currency":
            "MAGIC_CRYSTAL",

        "source":
            "BOSS_DROP",

        "boss_id":
            spec["boss_id"],

        "set_id":
            spec["set_id"],

        "unique":
            False,
    }


def main():
    legendary = read_json(
        LEGENDARY_FILE
    )

    mythic = read_json(
        MYTHIC_FILE
    )

    sets = read_json(
        SETS_FILE
    )

    bosses = read_json(
        BOSSES_FILE
    )

    boss_by_set = {
        boss["set_id"]: boss
        for boss in bosses
        if boss.get("set_id")
    }

    expected_set_ids = {
        spec["set_id"]
        for spec in SPECS
    }

    expected_item_ids = {
        f"{spec['set_id']}_{suffix}"
        for spec in SPECS
        for suffix in (
            "weapon",
            "shield",
            "armor",
        )
    }

    current_set_ids = {
        entry["id"]
        for entry in sets
    }

    current_item_ids = {
        item["id"]
        for item in legendary + mythic
    }

    present_sets = (
        expected_set_ids &
        current_set_ids
    )

    present_items = (
        expected_item_ids &
        current_item_ids
    )

    if (
        present_sets == expected_set_ids
        and
        present_items == expected_item_ids
    ):
        print(
            "✅ Todos os 14 sets e 42 itens já existem."
        )
        return

    if present_sets or present_items:
        raise SystemExit(
            "ERRO: geração parcial detectada. "
            "Restaure os backups antes de executar novamente.\n"
            f"Sets já presentes: {len(present_sets)}/14\n"
            f"Itens já presentes: {len(present_items)}/42"
        )

    new_legendary = []
    new_mythic = []
    new_sets = []

    for spec in SPECS:
        boss = boss_by_set.get(
            spec["set_id"]
        )

        if not boss:
            raise RuntimeError(
                f"Boss não encontrado para set "
                f"{spec['set_id']}"
            )

        if boss["id"] != spec["boss_id"]:
            raise RuntimeError(
                f"boss_id divergente em "
                f"{spec['set_id']}: "
                f"{boss['id']} != {spec['boss_id']}"
            )

        if boss["rarity"] != spec["rarity"]:
            raise RuntimeError(
                f"raridade divergente em "
                f"{spec['set_id']}"
            )

        spec["_boss_name"] = boss["name"]

        pieces = [
            create_item(
                spec,
                "weapon",
                "WEAPON",
                spec["weapon_name"],
                spec["weapon"],
            ),
            create_item(
                spec,
                "shield",
                "SHIELD",
                spec["shield_name"],
                spec["shield"],
            ),
            create_item(
                spec,
                "armor",
                "ARMOR",
                spec["armor_name"],
                spec["armor"],
            ),
        ]

        target = (
            new_legendary
            if spec["rarity"] == "LEGENDARY"
            else new_mythic
        )

        target.extend(
            pieces
        )

        new_sets.append(
            {
                "id":
                    spec["set_id"],

                "name":
                    spec["set_name"],

                "rarity":
                    spec["rarity"],

                "theme":
                    spec["theme"],

                "pieces":
                    [
                        item["id"]
                        for item in pieces
                    ],

                "bonuses":
                    spec["bonuses"],
            }
        )

    if len(new_legendary) != 24:
        raise RuntimeError(
            f"esperados 24 Legendary, "
            f"gerados {len(new_legendary)}"
        )

    if len(new_mythic) != 18:
        raise RuntimeError(
            f"esperados 18 Mythic, "
            f"gerados {len(new_mythic)}"
        )

    if len(new_sets) != 14:
        raise RuntimeError(
            f"esperados 14 sets, "
            f"gerados {len(new_sets)}"
        )

    final_legendary = (
        legendary +
        new_legendary
    )

    final_mythic = (
        mythic +
        new_mythic
    )

    final_sets = (
        sets +
        new_sets
    )

    all_item_ids = [
        item["id"]
        for item in
        final_legendary +
        final_mythic
    ]

    if len(all_item_ids) != len(
        set(all_item_ids)
    ):
        raise RuntimeError(
            "IDs de itens duplicados após geração"
        )

    all_set_ids = [
        entry["id"]
        for entry in final_sets
    ]

    if len(all_set_ids) != len(
        set(all_set_ids)
    ):
        raise RuntimeError(
            "IDs de sets duplicados após geração"
        )

    item_lookup = {
        item["id"]: item
        for item in
        final_legendary +
        final_mythic
    }

    for item_set in new_sets:
        if len(
            item_set["pieces"]
        ) != 3:
            raise RuntimeError(
                f"set {item_set['id']} "
                f"não possui 3 peças"
            )

        for item_id in item_set["pieces"]:
            item = item_lookup.get(
                item_id
            )

            if not item:
                raise RuntimeError(
                    f"peça inexistente: {item_id}"
                )

            if item["set_id"] != item_set["id"]:
                raise RuntimeError(
                    f"relação inválida: "
                    f"{item_id} -> "
                    f"{item['set_id']}"
                )

    timestamp = datetime.now().strftime(
        "%Y%m%d-%H%M%S"
    )

    for path in (
        LEGENDARY_FILE,
        MYTHIC_FILE,
        SETS_FILE,
    ):
        backup = Path(
            str(path) +
            f".before-raid-sets-{timestamp}"
        )

        shutil.copy2(
            path,
            backup,
        )

        print(
            f"Backup: {backup}"
        )

    write_json_atomic(
        LEGENDARY_FILE,
        final_legendary,
    )

    write_json_atomic(
        MYTHIC_FILE,
        final_mythic,
    )

    write_json_atomic(
        SETS_FILE,
        final_sets,
    )

    print()
    print(
        "========================================"
    )
    print(
        "✅ RAID BOSS SETS GERADOS"
    )
    print(
        "========================================"
    )
    print(
        f"Legendary adicionados: "
        f"{len(new_legendary)}"
    )
    print(
        f"Mythic adicionados: "
        f"{len(new_mythic)}"
    )
    print(
        f"Sets adicionados: "
        f"{len(new_sets)}"
    )
    print(
        f"Total de equipamentos novos: "
        f"{len(new_legendary) + len(new_mythic)}"
    )


if __name__ == "__main__":
    main()
