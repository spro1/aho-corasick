# aho-corasick

[![CI](https://github.com/spro1/aho-corasick/actions/workflows/ci.yml/badge.svg)](https://github.com/spro1/aho-corasick/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/spro1/aho-corasick.svg)](https://pkg.go.dev/github.com/spro1/aho-corasick)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[English](README.md) | **한국어**

의존성 없는 Go용 고속 [Aho-Corasick](https://ko.wikipedia.org/wiki/아호-코라식_알고리즘)
다중 패턴 문자열 검색 라이브러리입니다.

패턴 집합을 한 번 컴파일해 두면, 패턴 개수와 상관없이 텍스트를 한 번만 선형으로
훑어서 모든 패턴을 찾습니다.

## 특징

- **빠름**: 패턴을 바이트 동치류(byte class) 기반의 밀집 DFA로 컴파일하므로
  입력 바이트 하나당 테이블 조회 한 번이면 됩니다. 순회 중 메모리 할당이 없습니다.
- **세 가지 매칭 방식**: `Standard`(고전 방식, 겹치는 매칭 지원),
  `LeftmostFirst`(정규식 `a|ab`와 같은 방식), `LeftmostLongest`(POSIX식 최장 매칭).
- **ASCII 대소문자 무시** 매칭.
- **Go 이터레이터**(`iter.Seq[Match]`) 지원: `break`로 언제든 검색 중단.
- 고정 문자열 또는 콜백으로 **치환**.
- **`string` / `[]byte`** API 모두 제공 (문자열은 복사 없이 검색).
- **동시성 안전**: `Automaton`은 생성 후 불변입니다.
- 신뢰할 수 없는 패턴을 위한 **메모리 한도** 옵션(`WithMaxMemory`).
- UTF-8 텍스트에서도 안전한 **바이트 단위 오프셋**.
- 테이블 테스트, 무작위 차등 테스트, 단순 구현과 비교하는 퍼즈 테스트로 검증
  (커버리지 약 97%).

## 설치

```sh
go get github.com/spro1/aho-corasick
```

Go 1.27 이상이 필요하며, 표준 라이브러리 외의 의존성은 없습니다.

## 빠른 시작

```go
ac, err := ahocorasick.New([]string{"he", "she", "his", "hers"})
if err != nil {
	log.Fatal(err)
}

for m := range ac.IterOverlappingString("ushers") {
	fmt.Printf("%q at [%d:%d]\n", ac.Pattern(m.Pattern), m.Start, m.End)
}
// "she" at [1:4]
// "he" at [2:4]
// "hers" at [2:6]
```

## 사용법

### 매칭 방식

| 종류              | 겹치지 않는 검색에서 보고하는 매칭                         | 겹치는 검색 |
|-------------------|------------------------------------------------------------|:-----------:|
| `Standard`        | **가장 먼저 끝나는** 매칭 (고전 Aho-Corasick)              | ✅          |
| `LeftmostFirst`   | 가장 왼쪽 매칭, 시작 위치가 같으면 먼저 등록한 패턴 우선   | –           |
| `LeftmostLongest` | 가장 왼쪽 매칭, 시작 위치가 같으면 가장 긴 패턴 우선       | –           |

```go
patterns := []string{"Sam", "Samwise"}
ahocorasick.MustNew(patterns).FindString("Samwise") // "Sam"
ahocorasick.MustNew(patterns, ahocorasick.WithMatchKind(ahocorasick.LeftmostLongest)).
	FindString("Samwise") // "Samwise"
```

토큰화, 하이라이트, 치환 용도라면 보통 `LeftmostFirst`나 `LeftmostLongest`가 가장
직관적인 결과를 줍니다.

### 검색

```go
ac := ahocorasick.MustNew([]string{"error", "warn"}, ahocorasick.WithASCIICaseInsensitive(true))

ac.IsMatchString(line)            // bool, 첫 매칭에서 즉시 반환
ac.FindString(line)               // (Match, bool), 첫 매칭
ac.FindAllString(line)            // []Match, 겹치지 않는 모든 매칭
ac.FindAllOverlappingString(line) // []Match, 모든 출현 위치 (Standard 전용)

for m := range ac.IterString(line) { // 할당 없는 순회
	fmt.Println(line[m.Start:m.End])
}
```

`String` 접미사가 없는 메서드(`IsMatch`, `Find`, `FindAll`, `Iter` 등)는 `[]byte`를 받습니다.
`Match`의 `Start`, `End`는 바이트 오프셋이며 `End`는 포함하지 않습니다
(`haystack[Start:End]`가 매칭된 텍스트).

### 치환

```go
ac := ahocorasick.MustNew([]string{"apple", "banana"},
	ahocorasick.WithMatchKind(ahocorasick.LeftmostLongest))

ac.ReplaceAllString("apple and banana", []string{"사과", "바나나"})
// "사과 and 바나나"
```

### 옵션

| 옵션                              | 설명                                                        |
|-----------------------------------|-------------------------------------------------------------|
| `WithMatchKind(kind)`             | `Standard`(기본값), `LeftmostFirst`, `LeftmostLongest`      |
| `WithASCIICaseInsensitive(bool)`  | ASCII `A-Z` / `a-z` 대소문자 무시                           |
| `WithMaxMemory(bytes)`            | 오토마톤이 한도를 넘으면 `ErrTooLarge` 반환                 |

### 오류

`New`/`NewBytes`는 `ErrEmptyPattern`(빈 패턴), `ErrInvalidMatchKind`,
`ErrTooLarge`를 반환할 수 있으며 `errors.Is`로 판별합니다. `MustNew`는 오류 시
panic 합니다.

## 성능

1 MiB 무작위 소문자 텍스트 기준 검색 처리량 (Intel Xeon @ 2.10 GHz, Go 1.27.4):

| 패턴 수 | Standard | LeftmostFirst | LeftmostLongest | Overlapping |
|--------:|---------:|--------------:|----------------:|------------:|
| 10      | 464 MB/s | 450 MB/s      | 459 MB/s        | 439 MB/s    |
| 1,000   | 285 MB/s | 334 MB/s      | 328 MB/s        | 341 MB/s    |
| 10,000  | 181 MB/s | 188 MB/s      | 200 MB/s        | 207 MB/s    |

패턴 100개 기준으로 이전 버전(7.8 MB/s) 대비 약 59배(약 460 MB/s) 빠릅니다.

## 이전 버전에서 마이그레이션

```go
// 이전
transitions, outputs, fails := ahocorasick.MakeNode(patterns)
results := ahocorasick.Match(text, transitions, outputs, fails)

// 현재
ac, err := ahocorasick.New(patterns)
matches := ac.FindAllOverlappingString(text)
```

`Match.End`는 이제 배타적(exclusive)이며(이전 `EndIdx`는 포함), 모든 오프셋은
바이트 단위입니다. 이전 버전에 있던 버그(어떤 패턴이 다른 패턴의 접두사일 때 매칭
누락, 접미사를 공유하는 패턴이 여러 개일 때 하나만 보고)도 수정되었습니다.

## 라이선스

[Apache License 2.0](LICENSE)
