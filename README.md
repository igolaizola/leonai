# leonai 🦁🤖

**leonai** is an unofficial CLI tool for [Leonardo AI](https://app.leonardo.ai/)

> 📢 Connect with us! Join our Telegram group for support and collaboration: [t.me/igohub](https://t.me/igohub)

## 🚀 Features

- Video generation from image prompts

## 📦 Installation

You can use the Golang binary to install **leonai**:

```bash
go install github.com/igolaizola/leonai/cmd/leonai@latest
```

Or you can download the binary from the [releases](https://github.com/igolaizola/leonai/releases)

## 📋 Requirements

You need to capture the cookie from [Leonardo AI](https://app.leonardo.ai/) website.

1. Go to https://app.leonardo.ai/
2. Login if you are not already logged in
3. Open the developer tools (F12)
4. Go to the "Network" tab
5. Refresh the page
6. Click on the first request to https://app.leonardo.ai/api/auth/session
7. Go to the "Request Headers"
8. Copy the "cookie" header and save it in a file, e.g. `cookie.txt`

## 🕹️ Usage

Generate a video from an image prompt:

```bash
leonai generate --cookie cookie.txt --image car.jpg --output car.mp4 --motion-strength 5
```

### Help

Launch `leonai` with the `--help` flag to see all available commands and options:

```bash
leonai --help
```

You can use the `--help` flag with any command to view available options:

```bash
leonai video --help
```

### How to launch commands

Launch commands using a configuration file:

```bash
leonai video --config leonai.conf
```

```bash
# leonai.conf
cookie cookie.txt
image car.jpg
output car.mp4
motion-strength 5
```

Using environment variables (`LEONAI_` prefix, uppercase and underscores):

```bash
export LEONAI_COOKIE=cookie.txt
export LEONAI_IMAGE="car.jpg"
export LEONAI_OUTPUT="car.mp4"
export LEONAI_MOTION_STRENGTH=5
leonai video
```

Using command line arguments:

```bash
leonai video --cookie cookie.txt --image car.jpg --output car.mp4 --motion-strength 5
```

## ⚠️ Disclaimer

The automation of LeonardoAI accounts is a violation of their Terms of Service and will result in your account(s) being terminated.

Read about LeonardoAI Terms of Service and Community Guidelines.

leonai was written as a proof of concept and the code has been released for educational purposes only. The authors are released of any liabilities which your usage may entail.

## 💖 Support

If you have found my code helpful, please give the repository a star ⭐

Additionally, if you would like to support my late-night coding efforts and the coffee that keeps me going, I would greatly appreciate a donation.

You can invite me for a coffee at ko-fi (0% fees):

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/igolaizola)

Or at buymeacoffee:

[![buymeacoffee](https://user-images.githubusercontent.com/11333576/223217083-123c2c53-6ab8-4ea8-a2c8-c6cb5d08e8d2.png)](https://buymeacoffee.com/igolaizola)

Donate to my PayPal:

[paypal.me/igolaizola](https://www.paypal.me/igolaizola)

Sponsor me on GitHub:

[github.com/sponsors/igolaizola](https://github.com/sponsors/igolaizola)

Or donate to any of my crypto addresses:

- BTC `bc1qvuyrqwhml65adlu0j6l59mpfeez8ahdmm6t3ge`
- ETH `0x960a7a9cdba245c106F729170693C0BaE8b2fdeD`
- USDT (TRC20) `TD35PTZhsvWmR5gB12cVLtJwZtTv1nroDU`
- USDC (BEP20) / BUSD (BEP20) `0x960a7a9cdba245c106F729170693C0BaE8b2fdeD`
- Monero `41yc4R9d9iZMePe47VbfameDWASYrVcjoZJhJHFaK7DM3F2F41HmcygCrnLptS4hkiJARCwQcWbkW9k1z1xQtGSCAu3A7V4`

Thanks for your support!
