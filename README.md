# WhatsApp Sticker Bot

Bot de WhatsApp desenvolvido em **Go (Golang)** utilizando a biblioteca **WhatsMeow**, com suporte à conversão de **imagens, GIFs e vídeos curtos em figurinhas (.webp)**.

O projeto utiliza **Docker** para facilitar a execução em qualquer ambiente Linux ou Windows com WSL.

---

## Funcionalidades

* Conversão de **imagens** em figurinhas
* Conversão de **GIFs** em figurinhas animadas
* Conversão de **vídeos curtos** em figurinhas animadas
* Redimensionamento automático para **512x512**
* Compressão otimizada para WhatsApp
* Suporte a **Docker**
* Armazenamento temporário automático dos arquivos recebidos

---

## Tecnologias utilizadas

* **Go 1.25+**
* **WhatsMeow**
* **FFmpeg**
* **Docker / Docker Compose**
* **SQLite**

---

## Estrutura do projeto

```text
whatsapp-sticker-bot/
├── cmd/
├── internal/
├── temp/
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

---

## Como executar localmente

### Pré-requisitos

* Go 1.25 ou superior
* FFmpeg instalado
* Docker (opcional)

### Clonar o repositório

```bash
git clone https://github.com/ViniX3/whatsapp-sticker-bot.git
cd whatsapp-sticker-bot
```

### Instalar dependências

```bash
go mod tidy
```

### Executar

```bash
go run .
```

Ao iniciar, um **QR Code** será exibido no terminal para autenticação no WhatsApp.

---

## Executando com Docker

### Build da imagem

```bash
docker build -t whatsapp-sticker-bot .
```

### Executar container

```bash
docker run -it --name stickerbot whatsapp-sticker-bot
```

---

## Uso

Envie uma **imagem, GIF ou vídeo curto** com o comando:

```text
!f
```

O bot responderá com a figurinha convertida automaticamente.

---

## Roadmap

### v1.0.0

* [x] Conversão de imagens em figurinha
* [x] Conversão de GIFs
* [x] Conversão de vídeos curtos
* [x] Transparência corrigida
* [x] Execução via Docker

### v1.1.0

* [ ] Limitação de uso em grupos
* [ ] Controle de grupos permitidos
* [ ] Configuração de permissões básicas

### v1.2.0

* [ ] Sistema de usuários validados
* [ ] Lista de permissões
* [ ] Persistência de usuários autorizados

### v1.3.0

* [ ] Sistema de jogos interativos
* [ ] Pontuação por usuário
* [ ] Ranking simples

### v2.0.0

* [ ] Arquitetura focada em grupos específicos
* [ ] Sistema completo de jogos multiplayer
* [ ] Gerenciamento avançado de sessões
* [ ] Persistência de progresso e rankings

---

## Versionamento

Este projeto segue o padrão **Semantic Versioning (SemVer)**.

Exemplo:

```text
MAJOR.MINOR.PATCH
```

* **MAJOR** → mudanças incompatíveis
* **MINOR** → novas funcionalidades
* **PATCH** → correções e melhorias internas

Tags disponíveis:

```bash
git tag
```

---

## Publicando uma nova versão

```bash
git add .
git commit -m "feat: nova funcionalidade"
git tag -a v1.1.0 -m "Descrição da versão"
git push origin main --tags
```

---

## Autor

**Vinícius Barbosa**

* LinkedIn: https://www.linkedin.com/in/vinix3
* GitHub: https://github.com/ViniX3

---

## Licença

Este projeto está sob a licença **MIT**.

Sinta-se livre para estudar, modificar e compartilhar o código.
