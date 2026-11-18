# Docker Compose 로 croffle-server 띄우기

English: [README.md](./README.md)

비밀 키를 처음부터 넣고 띄우는 첫 배포입니다. 이 폴더에서 실행합니다.

```bash
# 1. 비밀 키. 나만 읽을 수 있게 만듭니다. 데이터 볼륨과 따로 사본을 두십시오.
#    키를 잃으면 일회용 코드를 쓰는 계정이 로그인하지 못합니다.
umask 077
echo "CROFFLE_SECRET_KEY=$(docker run --rm ghcr.io/team-croffle/croffle-server:0.2.0-rc.1 secret generate)" > croffle-secret.env

# 2. 시작. 로그에 "secret key: encrypted 0 one-time code secrets" 가 나옵니다.
docker compose up -d
docker compose logs croffle-server | grep 'secret key'

# 3. 첫 계정을 운영자로 만듭니다(비밀번호는 화면에 보이지 않게 입력).
docker compose exec croffle-server croffle-server user create <나> --password-stdin --operator
```

그다음 `http://<호스트>:8787` 에 로그인하면 운영 탭(Accounts, All devices,
Manage teams, Server)에서 나머지를 할 수 있습니다. 데스크톱 앱에서는
**설정 → 동기화** 에 같은 URL 을 넣고 로그인합니다.

키 파일이 아니라 env 파일을 쓰는 이유: 이미지는 권한 없는 사용자로 실행되므로,
호스트에서 마운트한 키 파일은 호스트의 모든 사용자가 읽을 수 있어야 컨테이너
안에서 읽힙니다. env 파일은 Compose 가 직접 읽으므로 파일을 나만 읽게 둘 수
있습니다. 설정 · 업그레이드 · 백업은 [서버 README](../README.ko.md) 를 보십시오.
