# Install Via Package installers

## Homebrew

Tap the burnt-labs/xion repository

```bash
brew tap burnt-labs/xion
```

Install xiond

```bash
brew install xiond
```

Verify Installation

```bash
xiond version
```

## Debian/Apt

Download the repository key

```bash
wget -qO - https://packages.burnt.com/apt/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/burnt-keyring.gpg
```

Add the burnt repository to your apt sources list, inlude the signing key

```bash
echo "deb [signed-by=/usr/share/keyrings/burnt-keyring.gpg] http://packages.burnt.com/apt /" | sudo tee /etc/apt/sources.list.d/burnt.list
```

Update sources, and install xiond

```bash
sudo apt update
sudo apt install xiond
```

Verify Installation

```bash
xiond version
```

## Redhat/Dnf/Yum/Rpm

Import the burnt repository key

```bash
sudo rpm --import https://packages.burnt.com/yum/gpg.key
```

Add the burnt repository to your repos list

```bash
printf "[burnt]\nname=Burnt Repo\nenabled=1\nbaseurl=https://packages.burnt.com/yum/\n" | sudo tee /etc/yum.repos.d/burnt.repo
```

Install xiond

```bash
sudo dnf install xiond
```

Verify Installation

```bash
xiond version
```

## Alpine Linux

Download the repository key

```bash
wget -qO - https://alpine.fury.io/burnt/burnt@fury.io-b8abd990.rsa.pub | sudo tee /etc/apk/keys/burnt@fury.io-b8abd990.rsa.pub 
```

Add the burnt repository to your repository list, inlude the signing key

```bash
echo "https://alpine.fury.io/burnt" >> /etc/apk/repositories
```

Update sources, and install xiond

```bash
sudo apk update
sudo apk add xiond
```

Verify Installation

```bash
xiond version
```
