export interface BuildLogDecoration {
  startLineNumber: number
  startColumn: number
  endLineNumber: number
  endColumn: number
  inlineClassName: string
}

export interface ParsedBuildLogAnsi {
  text: string
  decorations: BuildLogDecoration[]
}

const ANSI_ESCAPE_CHARACTER = String.fromCharCode(27)
const ANSI_ESCAPE_PATTERN = new RegExp(`${ANSI_ESCAPE_CHARACTER}\\[([0-9;]*)m`, "g")

const ANSI_FOREGROUND_CLASS_MAP = new Map<number, string>([
  [30, "build-log-ansi-fg-black"],
  [31, "build-log-ansi-fg-red"],
  [32, "build-log-ansi-fg-green"],
  [33, "build-log-ansi-fg-yellow"],
  [34, "build-log-ansi-fg-blue"],
  [35, "build-log-ansi-fg-magenta"],
  [36, "build-log-ansi-fg-cyan"],
  [37, "build-log-ansi-fg-white"],
  [90, "build-log-ansi-fg-bright-black"],
  [91, "build-log-ansi-fg-bright-red"],
  [92, "build-log-ansi-fg-bright-green"],
  [93, "build-log-ansi-fg-bright-yellow"],
  [94, "build-log-ansi-fg-bright-blue"],
  [95, "build-log-ansi-fg-bright-magenta"],
  [96, "build-log-ansi-fg-bright-cyan"],
  [97, "build-log-ansi-fg-bright-white"],
])

export function parseBuildLogAnsi(input: string): ParsedBuildLogAnsi {
  let activeClassName: string | null = null
  let output = ""
  const decorations: BuildLogDecoration[] = []

  let lineNumber = 1
  let columnNumber = 1
  let segmentStartLine: number | null = null
  let segmentStartColumn: number | null = null
  let cursor = 0

  const flushSegment = () => {
    if (
      activeClassName === null ||
      segmentStartLine === null ||
      segmentStartColumn === null ||
      (segmentStartLine === lineNumber && segmentStartColumn === columnNumber)
    ) {
      segmentStartLine = null
      segmentStartColumn = null
      return
    }

    decorations.push({
      startLineNumber: segmentStartLine,
      startColumn: segmentStartColumn,
      endLineNumber: lineNumber,
      endColumn: columnNumber,
      inlineClassName: activeClassName,
    })
    segmentStartLine = null
    segmentStartColumn = null
  }

  const startSegmentIfNeeded = () => {
    if (activeClassName === null || segmentStartLine !== null || segmentStartColumn !== null) {
      return
    }
    segmentStartLine = lineNumber
    segmentStartColumn = columnNumber
  }

  for (const match of input.matchAll(ANSI_ESCAPE_PATTERN)) {
    const matchIndex = match.index ?? 0
    appendVisibleText(input.slice(cursor, matchIndex))
    flushSegment()
    activeClassName = resolveForegroundClass(match[1] ?? "", activeClassName)
    cursor = matchIndex + match[0].length
  }

  appendVisibleText(input.slice(cursor))
  flushSegment()

  return {
    text: output,
    decorations,
  }

  function appendVisibleText(chunk: string) {
    for (const char of chunk) {
      if (char === "\r") {
        continue
      }

      if (char === "\n") {
        flushSegment()
        output += char
        lineNumber += 1
        columnNumber = 1
        continue
      }

      startSegmentIfNeeded()
      output += char
      columnNumber += 1
    }
  }
}

function resolveForegroundClass(rawCodes: string, currentClassName: string | null): string | null {
  const codes = rawCodes === "" ? [0] : rawCodes.split(";").map((value) => Number.parseInt(value, 10))
  let nextClassName = currentClassName

  for (let index = 0; index < codes.length; index += 1) {
    const code = codes[index]
    if (Number.isNaN(code)) {
      continue
    }

    if (code === 0 || code === 39) {
      nextClassName = null
      continue
    }

    if (code === 38 && codes[index + 1] === 5) {
      const paletteCode = codes[index + 2]
      nextClassName = resolveEightBitForegroundClass(paletteCode, nextClassName)
      index += 2
      continue
    }

    const mappedClassName = ANSI_FOREGROUND_CLASS_MAP.get(code)
    if (mappedClassName) {
      nextClassName = mappedClassName
    }
  }

  return nextClassName
}

function resolveEightBitForegroundClass(paletteCode: number | undefined, fallback: string | null): string | null {
  switch (paletteCode) {
    case 0:
      return "build-log-ansi-fg-black"
    case 1:
    case 9:
    case 196:
    case 197:
      return "build-log-ansi-fg-bright-red"
    case 2:
    case 10:
    case 46:
    case 47:
      return "build-log-ansi-fg-bright-green"
    case 3:
    case 11:
    case 226:
      return "build-log-ansi-fg-bright-yellow"
    case 4:
    case 12:
    case 33:
      return "build-log-ansi-fg-bright-blue"
    case 5:
    case 13:
    case 201:
      return "build-log-ansi-fg-bright-magenta"
    case 6:
    case 14:
    case 45:
    case 51:
      return "build-log-ansi-fg-bright-cyan"
    case 7:
    case 15:
      return "build-log-ansi-fg-bright-white"
    case 8:
    case 242:
    case 244:
      return "build-log-ansi-fg-bright-black"
    default:
      return fallback
  }
}
