# WhatsApp Sticker Bot

Bot para WhatsApp desenvolvido em **Go (Golang)** utilizando a biblioteca **WhatsMeow**.

O projeto começou como uma ferramenta para transformar **imagens, GIFs e vídeos em figurinhas**, mas evoluiu para um bot completo voltado para grupos do WhatsApp, contando com **economia virtual, jogos, quiz, ranking, interações entre usuários e sistema de comandos**.

A versão atual, **v3.0.0**, representa a implementação do novo ecossistema de jogos e interações do bot.

---

## Funcionalidades

### 🎨 Criação de figurinhas

O bot converte diferentes tipos de mídia para o formato de figurinha utilizado pelo WhatsApp.

Suporta:

* Imagens
* GIFs
* Vídeos curtos
* Mídias enviadas diretamente
* Mídias respondidas através de mensagens citadas
* Redimensionamento automático para **512x512**
* Conversão utilizando **FFmpeg + WebP**
* Processamento temporário automático dos arquivos

Comando:

```text
!f
```

---

## 💰 Sistema Gold

O bot possui uma economia virtual baseada em **Gold**.

Cada jogador pode possuir uma carteira e utilizar seu saldo nos diferentes jogos disponíveis.

As carteiras são separadas por grupo, permitindo que cada grupo tenha sua própria economia e ranking.

### Comandos

```text
!gold
```

Cria a carteira Gold do jogador e concede o bônus inicial, caso ainda não tenha sido resgatado.

```text
!saldo
```

Exibe o saldo atual de Gold.

```text
!bet <valor>
```

Realiza uma aposta utilizando uma quantidade de Gold.

Exemplo:

```text
!bet 500
```

O sistema possui diferentes resultados e multiplicadores, incluindo prêmios especiais e jackpots.

```text
!pix @pessoa <valor>
```

Transfere Gold para outro jogador do grupo.

Exemplo:

```text
!pix @pessoa 1000
```

```text
!roubar @pessoa
```

Tenta roubar uma porcentagem do Gold de outro jogador.

A tentativa pode resultar em sucesso ou penalidade para quem tentou realizar o roubo.

```text
!escudo
```

Compra uma proteção temporária contra tentativas de roubo.

```text
!sorte
```

Participa do sorteio diário e pode receber diferentes quantidades de Gold de acordo com a raridade sorteada.

```text
!ranking
```

Exibe o ranking dos jogadores com maior quantidade de Gold dentro do grupo.

---

## 🧠 Sistema de Quiz

O comando:

```text
!quiz
```

inicia uma pergunta aleatória para o jogador.

O banco atual possui:

```text
600 perguntas
```

distribuídas entre seis níveis de dificuldade:

```text
🟢 SUPER FÁCIL
🔵 FÁCIL
🟡 MÉDIO
🟠 DIFÍCIL
🔴 SUPER DIFÍCIL
💀 INSANO
```

Cada dificuldade possui **100 perguntas** de diferentes áreas do conhecimento, incluindo:

* História
* Geografia
* Ciências
* Matemática
* Lógica
* Tecnologia
* Computação
* Redes
* Biologia
* Astronomia
* Literatura
* Artes
* Música
* Cinema
* Games
* Esportes
* Cultura geral

As respostas são enviadas utilizando:

```text
A
B
C
D
```

Somente o jogador que iniciou o quiz pode responder à pergunta.

Existe apenas **um quiz ativo por grupo por vez**.

### Tempo para resposta

Os níveis normais possuem:

```text
15 segundos
```

O nível:

```text
💀 INSANO
```

possui apenas:

```text
6 segundos
```

### Recompensas

Quanto maior a dificuldade, maior a quantidade de Gold recebida ao responder corretamente.

As recompensas podem chegar a:

```text
50.000 Gold
```

nas perguntas do nível Insano.

O sistema também mantém um histórico temporário das últimas perguntas utilizadas em cada grupo, evitando que as mesmas perguntas apareçam repetidamente em sequência.

---

## 🎉 Interações entre jogadores

O bot também possui comandos recreativos para interação dentro dos grupos.

### Beijo

```text
!beijo @pessoa
```

Envia uma interação de beijo para a pessoa marcada.

Também pode ser utilizado sem mencionar ninguém:

```text
!beijo
```

Nesse caso, o bot escolhe aleatoriamente outro participante do grupo.

Existem **15 mensagens diferentes** que podem ser sorteadas.

### Tapa

```text
!tapa @pessoa
```

Envia uma interação de tapa para a pessoa marcada.

Também pode ser utilizado sem mencionar ninguém:

```text
!tapa
```

Nesse caso, uma pessoa do grupo é escolhida aleatoriamente.

Também existem **15 variações diferentes** de mensagem.

O próprio autor do comando e a conta utilizada pelo bot são excluídos do sorteio automático.

---

## 📖 Menu de comandos

O comando:

```text
!menu
```

exibe todos os comandos disponíveis diretamente no WhatsApp.

Isso permite que os usuários consultem rapidamente as funcionalidades do bot sem precisar acessar documentação externa.

---

## ❓ Identificação de comandos inexistentes

Todas as funcionalidades do bot utilizam o padrão:

```text
!comando
```

Caso um usuário tente executar um comando que não existe, por exemplo:

```text
!roubo
!roubargold
!quizzzz
```

o bot identifica automaticamente a tentativa e responde:

```text
@usuario esse comando não existe, use !menu para verificar os comandos que existem!
```

Mensagens normais que não começam com `!` continuam sendo ignoradas pelo sistema de comandos.

---

## 🔐 Controle de grupos

O bot pode ser limitado a grupos autorizados.

Os grupos permitidos são definidos através da configuração:

```text
groups.json
```

Mensagens enviadas em grupos que não fazem parte da lista de autorização são ignoradas.

Isso permite utilizar a mesma instância do WhatsApp em diferentes grupos sem necessariamente habilitar o bot em todos eles.

---

## ⚙️ Processamento

O bot utiliza processamento assíncrono para lidar com as mensagens recebidas.

As tarefas de conversão de mídia são processadas através de workers, evitando que operações mais pesadas, como conversão de vídeos e GIFs com FFmpeg, bloqueiem o processamento das demais mensagens.

---

## Tecnologias utilizadas

```text
Go 1.26
WhatsMeow
FFmpeg
SQLite
Docker
Docker Compose
```

### Go

Utilizado como linguagem principal da aplicação.

### WhatsMeow

Biblioteca responsável pela comunicação com o WhatsApp.

### FFmpeg

Responsável pelo processamento e conversão de imagens, vídeos e GIFs para o formato utilizado pelas figurinhas.

### SQLite

Responsável pela persistência da economia Gold, jogadores, transações e demais informações relacionadas aos jogos.

### Docker

Utilizado para empacotar e executar toda a aplicação de forma padronizada.

---

## Estrutura do projeto

```text
whatsapp-sticker-bot/
├── cmd/
│   └── bot/
│
├── internal/
│   ├── album/
│   ├── auth/
│   ├── config/
│   ├── converter/
│   ├── database/
│   ├── gold/
│   ├── handler/
│   ├── logger/
│   ├── media/
│   ├── processor/
│   ├── quiz/
│   │   └── questions/
│   └── whatsapp/
│
├── sessions/
├── storage/
├── temp/
├── groups.json
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

---

## Banco de dados

O projeto utiliza **SQLite** para armazenar os dados persistentes do sistema.

Entre as informações armazenadas estão:

```text
Usuários
Carteiras Gold
Saldos por grupo
Transações
Escudos
Sorte diária
```

O banco principal fica localizado em:

```text
storage/gold.db
```

O SQLite é utilizado em modo **WAL**, proporcionando melhor comportamento para operações concorrentes realizadas pelo bot.

---

## Banco de perguntas

As perguntas utilizadas pelo `!quiz` ficam armazenadas localmente em arquivos JSON:

```text
internal/quiz/questions/
├── super_easy.json
├── easy.json
├── medium.json
├── hard.json
├── super_hard.json
└── insane.json
```

Cada arquivo possui:

```text
100 perguntas
```

Total:

```text
600 perguntas
```

Os arquivos são incorporados ao binário através de `go:embed`, portanto não existe dependência de APIs externas para executar o quiz.

---

## Como executar localmente

### Pré-requisitos

É necessário possuir:

```text
Go 1.26+
FFmpeg
Git
```

Clone o repositório:

```bash
git clone https://github.com/ViniX3/whatsapp-sticker-bot.git

cd whatsapp-sticker-bot
```

Instale ou atualize as dependências:

```bash
go mod tidy
```

Execute:

```bash
go run ./cmd/bot
```

Na primeira autenticação, um **QR Code** será apresentado para conectar a conta do WhatsApp utilizada pelo bot.

Após a autenticação, os dados da sessão são armazenados para permitir reconexões futuras.

---

## Executando com Docker

O método recomendado para execução é através do Docker Compose.

### Build

```bash
docker compose build
```

### Iniciar

```bash
docker compose up -d
```

### Acompanhar logs

```bash
docker compose logs -f
```

### Parar

```bash
docker compose down
```

Para reconstruir completamente a imagem:

```bash
docker compose build --no-cache
docker compose up -d
```

---

## Testes

Antes de realizar um novo deploy, é possível validar todo o projeto com:

```bash
go test ./...
```

Também é possível validar o build completo:

```bash
CGO_ENABLED=1 go build \
  -o /tmp/whatsapp-sticker-bot-test \
  ./cmd/bot
```

---

## Comandos disponíveis

```text
🎨 Figurinhas
!f

💰 Gold
!gold
!saldo
!bet
!pix
!roubar
!escudo
!sorte
!ranking

🧠 Jogos
!quiz

🎉 Interações
!beijo
!tapa

📖 Ajuda
!menu
```

Atualmente o bot possui **13 comandos principais**.

---

## Versionamento

Este projeto utiliza **Semantic Versioning (SemVer)**:

```text
MAJOR.MINOR.PATCH
```

Onde:

```text
MAJOR  → grandes mudanças ou novas gerações do projeto
MINOR  → novas funcionalidades compatíveis
PATCH  → correções e melhorias internas
```

Para visualizar as versões disponíveis:

```bash
git tag
```

---

## Histórico de versões

### v1.x

Primeira geração do projeto.

Principais funcionalidades:

```text
Conversão de imagens em figurinhas
Conversão de GIFs
Conversão de vídeos
Processamento com FFmpeg
Execução através de Docker
```

### v2.x

Evolução da arquitetura e preparação do bot para utilização contínua em grupos.

Incluiu melhorias de:

```text
Estrutura interna
Processamento
Persistência
Controle de grupos
Estabilidade
Automação de atualização
```

### v3.0.0

Nova geração do projeto voltada para **jogos e interação dentro de grupos**.

Inclui:

```text
Sistema completo de economia Gold
Carteiras separadas por grupo
Sistema de apostas
Transferências entre jogadores
Roubo de Gold
Escudos
Sorte diária
Ranking
Quiz com 600 perguntas
6 níveis de dificuldade
Recompensas em Gold
Anti-repetição de perguntas
Comandos !beijo e !tapa
Seleção aleatória de participantes
Menu interativo
Identificação de comandos inexistentes
```

---

## Publicando uma nova versão

Adicione as alterações:

```bash
git add .
```

Crie o commit:

```bash
git commit -m "feat: nova funcionalidade"
```

Envie para o repositório:

```bash
git push origin main
```

Crie uma tag:

```bash
git tag -a v3.0.0 -m "Descrição da versão"
```

Envie a tag:

```bash
git push origin v3.0.0
```

Ou envie todas as tags locais:

```bash
git push origin --tags
```

---

## Autor

**Vinícius Barbosa**

LinkedIn:

https://www.linkedin.com/in/viniciusbarbosa2003

GitHub:

https://github.com/ViniX3

---

## Licença

Este projeto está sob a licença **MIT**.

Sinta-se livre para estudar, modificar e compartilhar o código.
