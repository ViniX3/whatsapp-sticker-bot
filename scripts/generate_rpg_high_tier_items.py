#!/usr/bin/env python3

import json
import shutil
import sys
from datetime import datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

RPG_DATA = (
    ROOT
    / "internal"
    / "rpg"
    / "data"
)

ITEM_DIR = (
    RPG_DATA
    / "items"
)

SETS_FILE = (
    RPG_DATA
    / "sets.json"
)

LEGENDARY_FILE = (
    ITEM_DIR
    / "legendary.json"
)

MYTHIC_FILE = (
    ITEM_DIR
    / "mythic.json"
)


def fail(message):
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
        fail(
            f"arquivo não encontrado: {path}"
        )

    except json.JSONDecodeError as exc:
        fail(
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


def make_item(
    item_id,
    name,
    item_type,
    rarity,
    power,
    attack,
    defense,
    description,
    boss_id,
    set_id="",
):
    if (
        attack
        + defense
        != power
    ):
        fail(
            f"atributos inconsistentes: {item_id}"
        )

    item = {
        "id": item_id,
        "name": name,
        "type": item_type,
        "rarity": rarity,
        "power": power,
        "attack": attack,
        "defense": defense,
        "description": description,
        "origin": (
            "Troféu de Boss"
        ),
        "value": 0,
        "currency": (
            "MAGIC_CRYSTAL"
        ),
        "source": (
            "BOSS_DROP"
        ),
        "boss_id": boss_id,
    }

    if set_id:
        item["set_id"] = (
            set_id
        )

    return item


def effect(
    bonus_type,
    value,
    description,
):
    return {
        "type": bonus_type,
        "value": value,
        "description": description,
    }


def make_set(
    set_id,
    name,
    rarity,
    theme,
    pieces,
    bonus_2,
    bonus_3,
):
    return {
        "id": set_id,
        "name": name,
        "rarity": rarity,
        "theme": theme,
        "pieces": pieces,
        "bonuses": [
            {
                "required_pieces": 2,
                "effects": bonus_2,
            },
            {
                "required_pieces": 3,
                "effects": bonus_3,
            },
        ],
    }


legendary_items = []
legendary_sets = []


def add_legendary_set(
    set_id,
    set_name,
    theme,
    boss_id,
    weapon,
    shield,
    armor,
    bonus_2,
    bonus_3,
):
    weapon_id = (
        f"{set_id}_weapon"
    )

    shield_id = (
        f"{set_id}_shield"
    )

    armor_id = (
        f"{set_id}_armor"
    )

    legendary_items.extend(
        [
            make_item(
                weapon_id,
                weapon[0],
                "WEAPON",
                "LEGENDARY",
                weapon[1],
                weapon[2],
                weapon[3],
                weapon[4],
                boss_id,
                set_id,
            ),
            make_item(
                shield_id,
                shield[0],
                "SHIELD",
                "LEGENDARY",
                shield[1],
                shield[2],
                shield[3],
                shield[4],
                boss_id,
                set_id,
            ),
            make_item(
                armor_id,
                armor[0],
                "ARMOR",
                "LEGENDARY",
                armor[1],
                armor[2],
                armor[3],
                armor[4],
                boss_id,
                set_id,
            ),
        ]
    )

    legendary_sets.append(
        make_set(
            set_id,
            set_name,
            "LEGENDARY",
            theme,
            [
                weapon_id,
                shield_id,
                armor_id,
            ],
            bonus_2,
            bonus_3,
        )
    )


add_legendary_set(
    "legendary_black_dragon",
    "Regalia do Dragão Negro",
    (
        "Equipamentos formados pelas "
        "escamas e pelo poder destrutivo "
        "do lendário Dragão Negro."
    ),
    "black_dragon",
    (
        "Espada do Dragão Negro",
        1260,
        1100,
        160,
        (
            "Uma espada negra cuja lâmina "
            "parece absorver a luz ao redor."
        ),
    ),
    (
        "Égide das Escamas Negras",
        1140,
        140,
        1000,
        (
            "Um escudo criado a partir das "
            "escamas mais resistentes do "
            "Dragão Negro."
        ),
    ),
    (
        "Armadura do Dragão Negro",
        1210,
        270,
        940,
        (
            "Uma couraça envolvida pela aura "
            "sombria da criatura."
        ),
    ),
    [
        effect(
            "BOSS_DAMAGE_PERCENT",
            8,
            "+8% de dano contra Bosses.",
        ),
    ],
    [
        effect(
            "DROP_CHANCE_PERCENT",
            6,
            "+6% de chance de drops.",
        ),
        effect(
            "COMBAT_POWER_PERCENT",
            5,
            "+5% de Poder de Combate.",
        ),
    ],
)

add_legendary_set(
    "legendary_leviathan",
    "Tesouros do Leviatã",
    (
        "Relíquias retiradas das profundezas "
        "dominadas pelo Leviatã."
    ),
    "leviathan",
    (
        "Tridente do Leviatã",
        1250,
        1080,
        170,
        (
            "Um tridente que carrega a força "
            "esmagadora dos oceanos."
        ),
    ),
    (
        "Escudo das Marés Abissais",
        1150,
        150,
        1000,
        (
            "Uma barreira coberta por símbolos "
            "das profundezas."
        ),
    ),
    (
        "Couraça do Abismo",
        1200,
        250,
        950,
        (
            "Armadura criada com escamas de "
            "criaturas que nunca viram a luz."
        ),
    ),
    [
        effect(
            "DUNGEON_SUCCESS_PERCENT",
            7,
            "+7% de chance de sucesso em Dungeons.",
        ),
    ],
    [
        effect(
            "RARE_DROP_CHANCE_PERCENT",
            6,
            "+6% de chance de drops raros.",
        ),
        effect(
            "CRYSTAL_REWARD_PERCENT",
            8,
            "+8% de Cristais Mágicos.",
        ),
    ],
)

add_legendary_set(
    "legendary_ancient_demon",
    "Arsenal do Demônio Ancestral",
    (
        "Equipamentos marcados pelas chamas "
        "do mais antigo dos demônios."
    ),
    "ancient_demon",
    (
        "Lâmina do Demônio Ancestral",
        1270,
        1110,
        160,
        (
            "A lâmina permanece quente mesmo "
            "longe das chamas infernais."
        ),
    ),
    (
        "Escudo dos Sete Selos",
        1140,
        160,
        980,
        (
            "Sete inscrições impedem que a "
            "energia demoníaca escape."
        ),
    ),
    (
        "Armadura do Inferno Antigo",
        1220,
        300,
        920,
        (
            "Placas queimadas pelas primeiras "
            "chamas do mundo inferior."
        ),
    ),
    [
        effect(
            "COMBAT_POWER_PERCENT",
            6,
            "+6% de Poder de Combate.",
        ),
    ],
    [
        effect(
            "BOSS_DAMAGE_PERCENT",
            8,
            "+8% de dano contra Bosses.",
        ),
        effect(
            "BLESSING_CHANCE_PERCENT",
            2,
            "2% de chance de receber uma bênção.",
        ),
    ],
)

add_legendary_set(
    "legendary_behemoth",
    "Armadura do Behemoth",
    (
        "Equipamentos pesados produzidos a "
        "partir da couraça do Behemoth."
    ),
    "behemoth",
    (
        "Machado do Behemoth",
        1240,
        1050,
        190,
        (
            "Um machado colossal feito para "
            "golpes capazes de romper muralhas."
        ),
    ),
    (
        "Baluarte da Carapaça Colossal",
        1170,
        110,
        1060,
        (
            "Um escudo quase impossível de "
            "movimentar por guerreiros comuns."
        ),
    ),
    (
        "Couraça do Colosso Selvagem",
        1230,
        200,
        1030,
        (
            "Proteção criada com placas da "
            "carapaça natural do Behemoth."
        ),
    ),
    [
        effect(
            "COMBAT_POWER_PERCENT",
            7,
            "+7% de Poder de Combate.",
        ),
    ],
    [
        effect(
            "DUNGEON_SUCCESS_PERCENT",
            6,
            "+6% de chance de sucesso em Dungeons.",
        ),
    ],
)

add_legendary_set(
    "legendary_storm_griffin",
    "Regalia do Grifo Tempestuoso",
    (
        "Relíquias do senhor alado das "
        "tempestades montanhosas."
    ),
    "storm_griffin",
    (
        "Lança do Grifo Tempestuoso",
        1230,
        1070,
        160,
        (
            "Uma lança que crepita com pequenas "
            "descargas elétricas."
        ),
    ),
    (
        "Égide dos Ventos Reais",
        1120,
        170,
        950,
        (
            "O ar ao redor deste escudo nunca "
            "permanece completamente imóvel."
        ),
    ),
    (
        "Armadura das Asas Trovejantes",
        1190,
        280,
        910,
        (
            "Uma armadura leve adornada com "
            "penas eletrificadas."
        ),
    ),
    [
        effect(
            "DROP_CHANCE_PERCENT",
            6,
            "+6% de chance de drops.",
        ),
    ],
    [
        effect(
            "GAME_LUCK_PERCENT",
            2,
            "+2% de Sorte.",
        ),
        effect(
            "RARE_DROP_CHANCE_PERCENT",
            4,
            "+4% de chance de drops raros.",
        ),
    ],
)

add_legendary_set(
    "legendary_frost_wyrm",
    "Tesouros da Serpente Glacial",
    (
        "Equipamentos preservados pelo frio "
        "sobrenatural da Serpente Glacial."
    ),
    "frost_wyrm",
    (
        "Garras da Serpente Glacial",
        1250,
        1100,
        150,
        (
            "Lâminas curvas que permanecem "
            "cobertas por gelo eterno."
        ),
    ),
    (
        "Escudo do Inverno Eterno",
        1140,
        130,
        1010,
        (
            "Um escudo cuja superfície nunca "
            "descongela."
        ),
    ),
    (
        "Armadura de Gelo Dracônico",
        1210,
        260,
        950,
        (
            "Escamas congeladas formam uma "
            "armadura de resistência extraordinária."
        ),
    ),
    [
        effect(
            "RARE_DROP_CHANCE_PERCENT",
            5,
            "+5% de chance de drops raros.",
        ),
    ],
    [
        effect(
            "DROP_RARITY_UPGRADE_CHANCE_PERCENT",
            3,
            "3% de chance de elevar a raridade de um drop.",
        ),
    ],
)

add_legendary_set(
    "legendary_infernal_colossus",
    "Arsenal do Colosso Infernal",
    (
        "Armas e armaduras criadas a partir "
        "do núcleo vulcânico do Colosso Infernal."
    ),
    "infernal_colossus",
    (
        "Martelo do Colosso Infernal",
        1280,
        1090,
        190,
        (
            "Cada golpe libera fragmentos "
            "incandescentes."
        ),
    ),
    (
        "Muralha de Magma",
        1150,
        140,
        1010,
        (
            "Um enorme escudo de rocha e magma "
            "solidificado."
        ),
    ),
    (
        "Armadura do Núcleo Vulcânico",
        1230,
        290,
        940,
        (
            "Fendas brilhantes atravessam as "
            "placas desta armadura."
        ),
    ),
    [
        effect(
            "BOSS_DAMAGE_PERCENT",
            7,
            "+7% de dano contra Bosses.",
        ),
    ],
    [
        effect(
            "CRYSTAL_REWARD_PERCENT",
            10,
            "+10% de Cristais Mágicos.",
        ),
    ],
)

add_legendary_set(
    "legendary_blood_moon_wolf",
    "Arsenal da Lua Sangrenta",
    (
        "Relíquias do Alfa que caça durante "
        "a rara Lua Sangrenta."
    ),
    "blood_moon_wolf",
    (
        "Presas da Lua Sangrenta",
        1240,
        1090,
        150,
        (
            "Duas lâminas curvas criadas das "
            "presas do Alfa Lunar."
        ),
    ),
    (
        "Escudo do Eclipse Lupino",
        1110,
        150,
        960,
        (
            "Um escudo negro marcado pela "
            "silhueta de uma lua vermelha."
        ),
    ),
    (
        "Armadura do Alfa Lunar",
        1180,
        280,
        900,
        (
            "Uma armadura leve impregnada "
            "pela essência do predador."
        ),
    ),
    [
        effect(
            "GAME_LUCK_PERCENT",
            2,
            "+2% de Sorte.",
        ),
    ],
    [
        effect(
            "DROP_CHANCE_PERCENT",
            7,
            "+7% de chance de drops.",
        ),
    ],
)

legendary_items.extend(
    [
        make_item(
            "legendary_forgotten_king_weapon",
            "Ceifadora do Rei Esquecido",
            "WEAPON",
            "LEGENDARY",
            1290,
            1140,
            150,
            (
                "A arma do soberano cujo nome "
                "foi apagado de todos os registros."
            ),
            "forgotten_king",
        ),
        make_item(
            "legendary_time_guardian_armor",
            "Armadura do Guardião do Tempo",
            "ARMOR",
            "LEGENDARY",
            1260,
            280,
            980,
            (
                "Algumas marcas em suas placas "
                "parecem envelhecer e desaparecer "
                "continuamente."
            ),
            "time_guardian",
        ),
    ]
)


mythic_items = []
mythic_sets = []


def add_mythic_set(
    set_id,
    set_name,
    theme,
    boss_id,
    weapon,
    shield,
    armor,
    bonus_2,
    bonus_3,
):
    weapon_id = (
        f"{set_id}_weapon"
    )

    shield_id = (
        f"{set_id}_shield"
    )

    armor_id = (
        f"{set_id}_armor"
    )

    mythic_items.extend(
        [
            make_item(
                weapon_id,
                weapon[0],
                "WEAPON",
                "MYTHIC",
                weapon[1],
                weapon[2],
                weapon[3],
                weapon[4],
                boss_id,
                set_id,
            ),
            make_item(
                shield_id,
                shield[0],
                "SHIELD",
                "MYTHIC",
                shield[1],
                shield[2],
                shield[3],
                shield[4],
                boss_id,
                set_id,
            ),
            make_item(
                armor_id,
                armor[0],
                "ARMOR",
                "MYTHIC",
                armor[1],
                armor[2],
                armor[3],
                armor[4],
                boss_id,
                set_id,
            ),
        ]
    )

    mythic_sets.append(
        make_set(
            set_id,
            set_name,
            "MYTHIC",
            theme,
            [
                weapon_id,
                shield_id,
                armor_id,
            ],
            bonus_2,
            bonus_3,
        )
    )


add_mythic_set(
    "mythic_phoenix",
    "Regalia da Fênix Eterna",
    (
        "Relíquias criadas a partir das "
        "chamas do eterno ciclo de renascimento."
    ),
    "phoenix",
    (
        "Lâmina da Fênix Eterna",
        1780,
        1570,
        210,
        (
            "Uma espada coberta por chamas "
            "que não queimam seu portador."
        ),
    ),
    (
        "Escudo das Cinzas Imortais",
        1620,
        190,
        1430,
        (
            "Cinzas da Fênix se recompõem "
            "sobre sua superfície."
        ),
    ),
    (
        "Armadura do Renascimento",
        1700,
        370,
        1330,
        (
            "Uma armadura quente ao toque "
            "que parece jamais se deteriorar."
        ),
    ),
    [
        effect(
            "CRYSTAL_REWARD_PERCENT",
            15,
            "+15% de Cristais Mágicos.",
        ),
    ],
    [
        effect(
            "BLESSING_CHANCE_PERCENT",
            4,
            "4% de chance de receber uma bênção.",
        ),
        effect(
            "DROP_CHANCE_PERCENT",
            8,
            "+8% de chance de drops.",
        ),
    ],
)

add_mythic_set(
    "mythic_ancient_titan",
    "Arsenal do Titã Primordial",
    (
        "Equipamentos formados com fragmentos "
        "do corpo de um Titã ancestral."
    ),
    "ancient_titan",
    (
        "Martelo do Titã Primordial",
        1840,
        1600,
        240,
        (
            "Seu peso seria impossível para "
            "qualquer guerreiro comum."
        ),
    ),
    (
        "Escudo dos Continentes",
        1680,
        160,
        1520,
        (
            "Uma muralha portátil feita de "
            "pedra primordial."
        ),
    ),
    (
        "Armadura do Titã Ancestral",
        1770,
        320,
        1450,
        (
            "Placas colossais comprimidas "
            "por força ancestral."
        ),
    ),
    [
        effect(
            "COMBAT_POWER_PERCENT",
            10,
            "+10% de Poder de Combate.",
        ),
    ],
    [
        effect(
            "BOSS_DAMAGE_PERCENT",
            12,
            "+12% de dano contra Bosses.",
        ),
    ],
)

add_mythic_set(
    "mythic_world_serpent",
    "Escamas da Serpente-Mundo",
    (
        "Relíquias obtidas da criatura que "
        "circunda terras e oceanos."
    ),
    "world_serpent",
    (
        "Presa da Serpente-Mundo",
        1810,
        1600,
        210,
        (
            "Uma arma curva produzida de uma "
            "presa colossal."
        ),
    ),
    (
        "Escudo das Escamas do Mundo",
        1660,
        180,
        1480,
        (
            "Escamas sobrepostas formam uma "
            "proteção praticamente impenetrável."
        ),
    ),
    (
        "Armadura da Serpente-Mundo",
        1740,
        350,
        1390,
        (
            "A superfície desta armadura "
            "se move como escamas vivas."
        ),
    ),
    [
        effect(
            "DROP_CHANCE_PERCENT",
            10,
            "+10% de chance de drops.",
        ),
    ],
    [
        effect(
            "DROP_RARITY_UPGRADE_CHANCE_PERCENT",
            4,
            "4% de chance de elevar a raridade de um drop.",
        ),
    ],
)

add_mythic_set(
    "mythic_celestial_hydra",
    "Regalia da Hidra Celestial",
    (
        "Equipamentos banhados pelo sangue "
        "regenerativo da Hidra Celestial."
    ),
    "celestial_hydra",
    (
        "Lâminas das Nove Cabeças",
        1800,
        1590,
        210,
        (
            "Um conjunto de lâminas que parece "
            "atacar de múltiplas direções."
        ),
    ),
    (
        "Égide da Hidra Celestial",
        1640,
        200,
        1440,
        (
            "Nove cabeças estão gravadas "
            "na superfície do escudo."
        ),
    ),
    (
        "Armadura do Sangue Regenerador",
        1730,
        380,
        1350,
        (
            "O material desta armadura parece "
            "recompor pequenas rachaduras."
        ),
    ),
    [
        effect(
            "DUNGEON_SUCCESS_PERCENT",
            10,
            "+10% de chance de sucesso em Dungeons.",
        ),
    ],
    [
        effect(
            "BLESSING_CHANCE_PERCENT",
            3,
            "3% de chance de receber uma bênção.",
        ),
        effect(
            "RARE_DROP_CHANCE_PERCENT",
            8,
            "+8% de chance de drops raros.",
        ),
    ],
)

add_mythic_set(
    "mythic_abyss_king",
    "Tesouros do Rei Abissal",
    (
        "Relíquias pertencentes ao soberano "
        "das regiões mais profundas do oceano."
    ),
    "abyss_king",
    (
        "Tridente do Rei Abissal",
        1830,
        1600,
        230,
        (
            "O símbolo máximo da autoridade "
            "sobre as profundezas."
        ),
    ),
    (
        "Escudo do Trono Afogado",
        1660,
        170,
        1490,
        (
            "Fragmentos do trono do Rei Abissal "
            "foram usados em sua criação."
        ),
    ),
    (
        "Armadura do Soberano das Profundezas",
        1760,
        360,
        1400,
        (
            "Água escura escorre continuamente "
            "por suas placas."
        ),
    ),
    [
        effect(
            "CRYSTAL_REWARD_PERCENT",
            15,
            "+15% de Cristais Mágicos.",
        ),
    ],
    [
        effect(
            "STELLAR_STONE_CHANCE_PERCENT",
            6,
            "+6% de chance de encontrar Pedras Estelares.",
        ),
    ],
)

add_mythic_set(
    "mythic_void_colossus",
    "Arsenal do Colosso do Vazio",
    (
        "Armas feitas de matéria arrancada "
        "de uma criatura parcialmente fora "
        "da realidade."
    ),
    "void_colossus",
    (
        "Espada da Fenda",
        1870,
        1660,
        210,
        (
            "Sua lâmina deixa pequenas fissuras "
            "escuras no ar ao ser movimentada."
        ),
    ),
    (
        "Escudo da Singularidade",
        1690,
        180,
        1510,
        (
            "Ataques parecem perder força ao "
            "aproximar-se de sua superfície."
        ),
    ),
    (
        "Armadura do Colosso do Vazio",
        1790,
        380,
        1410,
        (
            "Partes desta armadura desaparecem "
            "momentaneamente da realidade."
        ),
    ),
    [
        effect(
            "RARE_DROP_CHANCE_PERCENT",
            10,
            "+10% de chance de drops raros.",
        ),
    ],
    [
        effect(
            "GAME_LUCK_PERCENT",
            3,
            "+3% de Sorte.",
        ),
        effect(
            "DROP_RARITY_UPGRADE_CHANCE_PERCENT",
            4,
            "4% de chance de elevar a raridade de um drop.",
        ),
    ],
)

add_mythic_set(
    "mythic_eternal_lich",
    "Vestes do Lich Eterno",
    (
        "Artefatos alimentados pelas almas "
        "aprisionadas pelo Lich Eterno."
    ),
    "eternal_lich",
    (
        "Cetro do Lich Eterno",
        1790,
        1600,
        190,
        (
            "Um cetro que sussurra nomes de "
            "mortos esquecidos."
        ),
    ),
    (
        "Escudo das Almas Aprisionadas",
        1630,
        190,
        1440,
        (
            "Vozes podem ser ouvidas vindas "
            "de dentro do escudo."
        ),
    ),
    (
        "Armadura do Rei Morto",
        1720,
        390,
        1330,
        (
            "Uma armadura funerária revestida "
            "por energia necromântica."
        ),
    ),
    [
        effect(
            "DROP_CHANCE_PERCENT",
            9,
            "+9% de chance de drops.",
        ),
    ],
    [
        effect(
            "BLESSING_CHANCE_PERCENT",
            4,
            "4% de chance de receber uma bênção.",
        ),
        effect(
            "CRYSTAL_REWARD_PERCENT",
            12,
            "+12% de Cristais Mágicos.",
        ),
    ],
)

add_mythic_set(
    "mythic_chaos_dragon",
    "Regalia do Dragão do Caos",
    (
        "Equipamentos contaminados pela "
        "energia instável do Dragão do Caos."
    ),
    "chaos_dragon",
    (
        "Lâmina do Dragão do Caos",
        1900,
        1680,
        220,
        (
            "A forma e a cor da lâmina parecem "
            "mudar constantemente."
        ),
    ),
    (
        "Escudo das Escamas Caóticas",
        1700,
        200,
        1500,
        (
            "Escamas instáveis reorganizam-se "
            "após cada impacto."
        ),
    ),
    (
        "Armadura do Caos Primordial",
        1810,
        410,
        1400,
        (
            "Energia imprevisível percorre "
            "cada uma de suas placas."
        ),
    ),
    [
        effect(
            "BOSS_DAMAGE_PERCENT",
            10,
            "+10% de dano contra Bosses.",
        ),
    ],
    [
        effect(
            "COMBAT_POWER_PERCENT",
            10,
            "+10% de Poder de Combate.",
        ),
        effect(
            "DROP_RARITY_UPGRADE_CHANCE_PERCENT",
            4,
            "4% de chance de elevar a raridade de um drop.",
        ),
    ],
)

add_mythic_set(
    "mythic_moon_devourer",
    "Relíquias do Devorador de Luas",
    (
        "Artefatos provenientes da criatura "
        "que transforma luas em alimento."
    ),
    "moon_devourer",
    (
        "Foice do Devorador de Luas",
        1880,
        1660,
        220,
        (
            "A lâmina curva reflete um céu "
            "completamente sem luas."
        ),
    ),
    (
        "Égide da Lua Partida",
        1680,
        190,
        1490,
        (
            "Um fragmento lunar ocupa o centro "
            "deste escudo."
        ),
    ),
    (
        "Armadura do Céu Sem Lua",
        1780,
        390,
        1390,
        (
            "Uma armadura negra coberta por "
            "minúsculos pontos luminosos."
        ),
    ),
    [
        effect(
            "STELLAR_STONE_CHANCE_PERCENT",
            8,
            "+8% de chance de encontrar Pedras Estelares.",
        ),
    ],
    [
        effect(
            "GAME_LUCK_PERCENT",
            3,
            "+3% de Sorte.",
        ),
        effect(
            "DROP_CHANCE_PERCENT",
            10,
            "+10% de chance de drops.",
        ),
    ],
)

mythic_items.append(
    make_item(
        "mythic_celestial_emperor_weapon",
        "Lança do Imperador Celestial",
        "WEAPON",
        "MYTHIC",
        1920,
        1710,
        210,
        (
            "Uma lança concedida ao primeiro "
            "soberano capaz de desafiar os céus."
        ),
        "celestial_emperor",
    )
)


def validate():
    if len(
        legendary_items
    ) != 26:
        fail(
            "esperados 26 novos Lendários, "
            f"encontrados {len(legendary_items)}"
        )

    if len(
        mythic_items
    ) != 28:
        fail(
            "esperados 28 Míticos, "
            f"encontrados {len(mythic_items)}"
        )

    if len(
        legendary_sets
    ) != 8:
        fail(
            "esperados 8 novos sets Lendários"
        )

    if len(
        mythic_sets
    ) != 9:
        fail(
            "esperados 9 sets Míticos"
        )

    all_items = (
        legendary_items
        + mythic_items
    )

    ids = [
        item["id"]
        for item
        in all_items
    ]

    names = [
        item["name"]
        for item
        in all_items
    ]

    if len(ids) != len(
        set(ids)
    ):
        fail(
            "IDs duplicados entre itens altos"
        )

    if len(names) != len(
        set(names)
    ):
        fail(
            "nomes duplicados entre itens altos"
        )

    for item in all_items:
        if (
            item["attack"]
            + item["defense"]
            != item["power"]
        ):
            fail(
                "atributos inconsistentes: "
                f"{item['id']}"
            )

        if (
            item["source"]
            != "BOSS_DROP"
        ):
            fail(
                f"{item['id']} não é BOSS_DROP"
            )

        if not item.get(
            "boss_id"
        ):
            fail(
                f"{item['id']} sem boss_id"
            )


def update_sets():
    current = load_json(
        SETS_FILE
    )

    managed_ids = {
        item["id"]
        for item
        in (
            legendary_sets
            + mythic_sets
        )
    }

    current = [
        item
        for item
        in current
        if item.get(
            "id"
        ) not in managed_ids
    ]

    current.extend(
        legendary_sets
    )

    current.extend(
        mythic_sets
    )

    return current


def backup(
    path,
    stamp,
):
    if not path.exists():
        return

    shutil.copy2(
        path,
        path.with_name(
            path.name
            + ".before-high-tier-"
            + stamp
        ),
    )


def main():
    validate()

    ITEM_DIR.mkdir(
        parents=True,
        exist_ok=True,
    )

    stamp = (
        datetime.now()
        .strftime(
            "%Y%m%d-%H%M%S"
        )
    )

    for path in (
        LEGENDARY_FILE,
        MYTHIC_FILE,
        SETS_FILE,
    ):
        backup(
            path,
            stamp,
        )

    write_json(
        LEGENDARY_FILE,
        legendary_items,
    )

    write_json(
        MYTHIC_FILE,
        mythic_items,
    )

    write_json(
        SETS_FILE,
        update_sets(),
    )

    print(
        "Tiers altos gerados com sucesso."
    )

    print(
        "Novos Lendários: 26"
    )

    print(
        "Míticos:          28"
    )

    print(
        "Sets Lendários:    8"
    )

    print(
        "Sets Míticos:      9"
    )

    print(
        "Equipamentos finais esperados: 395"
    )


if __name__ == "__main__":
    main()
