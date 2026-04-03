/**
 * Prompt Template System
 *
 * Extensible module for building AI prompts. Each template exposes:
 *   id          – unique identifier
 *   name        – human-readable name (i18n key)
 *   description – short description (i18n key)
 *   build(channelNames: string[], userIntent?: string) => string
 */

// ---------------------------------------------------------------------------
// Template: Channel Alias Rules
// ---------------------------------------------------------------------------

const ALIAS_RULES_TEMPLATE = {
  id: 'alias_rules',
  name: 'rules.ai_template_alias_rules',
  description: 'rules.ai_template_alias_rules_desc',

  /**
   * Build a prompt that asks the AI to generate channel alias (rename) rules.
   *
   * @param {string[]} channelNames – deduplicated channel names
   * @param {string}   userIntent   – optional free-text description of desired naming format
   * @returns {string} the full prompt text
   */
  build(channelNames, userIntent = '') {
    const channelList = channelNames.join(', ')

    const intentSection = userIntent.trim()
      ? `
## User Requirements

The user has described the following naming preferences:

> ${userIntent.trim()}

Please generate alias rules that transform the channel names according to these requirements.`
      : `
## Auto-detect Mode

The user did not specify naming preferences. Please analyze the channel name list and automatically identify common naming inconsistencies that should be cleaned up, such as:
- Removing resolution suffixes (e.g. "高清", "HD", "4K", "1080P", "标清", "SD")
- Standardizing CCTV naming (e.g. "CCTV-1" → "CCTV1")
- Removing redundant whitespace or special characters
- Unifying channel name formats (e.g. "XX卫视频道" → "XX卫视")

Generate alias rules to normalize these channel names.`

    return `You are an IPTV channel management expert. I need you to generate a set of channel alias (rename) rules based on the provided channel name list.

## Task

Analyze the channel names below and generate regex-based alias rules to clean up and normalize the channel names.
${intentSection}

## Requirements

1. Each rule consists of a regex \`pattern\` for matching and a \`replacement\` string for the new name.
2. The \`replacement\` field uses Go's regexp replacement syntax: \`\${1}\`, \`\${2}\` to reference capture groups. The braces are REQUIRED.
3. Rules are matched in order; only the first matching rule takes effect for each channel.
4. Generate as few rules as possible while covering the most channels — prefer general patterns over one-rule-per-channel.
5. If a channel name does not need renaming, do NOT generate a rule for it.
6. The \`match_mode\` can be \`"regex"\` or \`"string"\`. Use \`"regex"\` for pattern-based matching and \`"string"\` for simple substring matching. When \`"string"\` is used, the \`replacement\` is the literal new name.

## Regex Syntax Constraints (CRITICAL)

The regex engine is **Go's regexp package (RE2)**. You MUST follow these rules:

- **DO NOT** use lookahead or lookbehind assertions: \`(?=...)\`, \`(?!...)\`, \`(?<=...)\`, \`(?<!...)\` are NOT supported.
- **DO NOT** use backreferences: \`\\1\`, \`\\2\`, etc. are NOT supported. Use \`\${1}\`, \`\${2}\` in the replacement string instead.
- **DO NOT** use possessive quantifiers: \`*+\`, \`++\`, \`?+\` are NOT supported.
- **DO NOT** use atomic groups: \`(?>...)\` is NOT supported.
- **Allowed syntax**: alternation \`|\`, character classes \`[...]\`, anchors \`^\` and \`$\`, quantifiers \`*\`, \`+\`, \`?\`, \`{n,m}\`, non-capturing groups \`(?:...)\`, and case-insensitive flag \`(?i)\`.

## Output Format

Return the result strictly in the following JSON format. Do not include any extra text or explanation — only output the raw JSON array:

\`\`\`json
[
  { "match_mode": "regex", "pattern": "regex_pattern", "replacement": "replacement_string" }
]
\`\`\`

## Example

### Input channel list:
CCTV-1 高清, CCTV-2 高清, CCTV-5+, 湖南卫视 HD, 浙江卫视高清, 东方卫视频道

### Output:
\`\`\`json
[
  { "match_mode": "regex", "pattern": "^CCTV-?(\\\\d+).*$", "replacement": "CCTV\${1}" },
  { "match_mode": "regex", "pattern": "\\\\s*(?:高清|HD|标清|SD)$", "replacement": "" },
  { "match_mode": "regex", "pattern": "^(.+卫视)频道$", "replacement": "\${1}" }
]
\`\`\`

## Now generate alias rules for the following channel list:

${channelList}
`
  },
}

// ---------------------------------------------------------------------------
// Template: Channel Filter Rules
// ---------------------------------------------------------------------------

const FILTER_RULES_TEMPLATE = {
  id: 'filter_rules',
  name: 'rules.ai_template_filter_rules',
  description: 'rules.ai_template_filter_rules_desc',

  /**
   * Build a prompt that asks the AI to generate channel filter rules.
   *
   * @param {string[]} channelNames – deduplicated channel names
   * @param {string}   userIntent   – REQUIRED free-text description of what to filter out
   * @returns {string} the full prompt text
   */
  build(channelNames, userIntent) {
    const channelList = channelNames.join(', ')

    return `You are an IPTV channel management expert. I need you to generate a set of channel filter rules based on the user's requirements and the provided channel name list.

## Task

Based on the user's filtering requirements, analyze the channel names below and generate rules that will **exclude (drop)** matching channels from the output.

## User Requirements

> ${userIntent.trim()}

## Requirements

1. Channels matching any filter rule will be **removed** from the output.
2. Each rule specifies a \`target\` (\`"name"\` to match the original name), a \`match_mode\` (\`"regex"\` or \`"string"\`), and a \`pattern\`.
3. Use \`"regex"\` mode for flexible pattern matching, and \`"string"\` mode for simple keyword matching (case-insensitive substring match).
4. The \`target\` should almost always be \`"name"\` (match against original channel name).
5. Generate as few rules as possible while accurately fulfilling the user's filtering requirements.
6. Only filter channels that exist in the provided list — do not invent rules for channels not present.

## Regex Syntax Constraints (CRITICAL)

The regex engine is **Go's regexp package (RE2)**. You MUST follow these rules:

- **DO NOT** use lookahead or lookbehind assertions: \`(?=...)\`, \`(?!...)\`, \`(?<=...)\`, \`(?<!...)\` are NOT supported.
- **DO NOT** use backreferences: \`\\1\`, \`\\2\`, etc. are NOT supported.
- **DO NOT** use possessive quantifiers: \`*+\`, \`++\`, \`?+\` are NOT supported.
- **DO NOT** use atomic groups: \`(?>...)\` is NOT supported.
- **Allowed syntax**: alternation \`|\`, character classes \`[...]\`, anchors \`^\` and \`$\`, quantifiers \`*\`, \`+\`, \`?\`, \`{n,m}\`, non-capturing groups \`(?:...)\`, and case-insensitive flag \`(?i)\`.

## Output Format

Return the result strictly in the following JSON format. Do not include any extra text or explanation — only output the raw JSON array:

\`\`\`json
[
  { "target": "name", "match_mode": "regex", "pattern": "regex_pattern" }
]
\`\`\`

## Example

### User requirement: "过滤掉所有购物频道和测试频道"

### Input channel list:
CCTV1, CCTV2, 湖南卫视, 东方购物, 家有购物, 测试频道1, 备用频道, 快乐购

### Output:
\`\`\`json
[
  { "target": "name", "match_mode": "regex", "pattern": "购物|快乐购" },
  { "target": "name", "match_mode": "regex", "pattern": "测试|备用" }
]
\`\`\`

## Now generate filter rules for the following channel list:

${channelList}
`
  },
}

// ---------------------------------------------------------------------------
// Template: Channel Group Rules
// ---------------------------------------------------------------------------

const GROUP_RULES_TEMPLATE = {
  id: 'group_rules',
  name: 'rules.ai_template_group_rules',
  description: 'rules.ai_template_group_rules_desc',

  /**
   * Build a prompt that asks the AI to generate channel grouping rules.
   * Uses Few-shot prompting with concrete input/output examples.
   *
   * @param {string[]} channelNames – deduplicated channel names
   * @param {string}   userIntent   – optional free-text description of desired grouping
   * @returns {string} the full prompt text
   */
  build(channelNames, userIntent = '') {
    const channelList = channelNames.join(', ')

    const intentSection = userIntent.trim()
      ? `

## User Requirements

The user has described the following grouping preferences:

> ${userIntent.trim()}

Please generate grouping rules according to these requirements.`
      : ''

    return `You are an IPTV channel management expert. I need you to generate a set of channel grouping rules based on the provided channel name list.

## Task

Analyze all the channel names below, categorize them into appropriate groups, and write regex matching rules for each group.
${intentSection}

## Requirements

1. No more than 7 groups in total.
2. Group names must be concise and descriptive.
3. **IMPORTANT: Group names must be in the same language as the channel names.** For example, if the channel names are in Chinese, the group names must also be in Chinese (e.g. 央视, 卫视, 地方, 国际). If the channel names are in English, use English group names.
4. Each group should contain one or more regex rules to match channel names.
5. Try to ensure every channel can be matched by at least one group's rules.
6. Regex patterns should be as concise and accurate as possible.

## Regex Syntax Constraints (CRITICAL)

The regex engine is **Go's regexp package (RE2)**. You MUST follow these rules:

- **DO NOT** use lookahead or lookbehind assertions: \`(?=...)\`, \`(?!...)\`, \`(?<=...)\`, \`(?<!...)\` are NOT supported.
- **DO NOT** use backreferences: \`\\1\`, \`\\2\`, etc. are NOT supported.
- **DO NOT** use possessive quantifiers: \`*+\`, \`++\`, \`?+\` are NOT supported.
- **DO NOT** use atomic groups: \`(?>...)\` is NOT supported.
- **Allowed syntax**: alternation \`|\`, character classes \`[...]\`, anchors \`^\` and \`$\`, quantifiers \`*\`, \`+\`, \`?\`, \`{n,m}\`, non-capturing groups \`(?:...)\`, and case-insensitive flag \`(?i)\`.
- Keep patterns simple: prefer plain text matching, alternation (\`A|B|C\`), and basic character classes.

## Output Format

Return the result strictly in the following JSON format. Do not include any extra text or explanation — only output the raw JSON array:

\`\`\`json
[
  {
    "group_name": "Group Name",
    "rules": [
      { "target": "name", "match_mode": "regex", "pattern": "regex_pattern" }
    ]
  }
]
\`\`\`

## Example

### Input channel list:
CCTV1, CCTV2, CCTV5, 湖南卫视, 浙江卫视, 北京卫视, 凤凰中文, 凤凰资讯, CGTN, 东方卫视

### Output:
\`\`\`json
[
  {
    "group_name": "央视",
    "rules": [
      { "target": "name", "match_mode": "regex", "pattern": "^CCTV" }
    ]
  },
  {
    "group_name": "卫视",
    "rules": [
      { "target": "name", "match_mode": "regex", "pattern": "卫视" }
    ]
  },
  {
    "group_name": "国际",
    "rules": [
      { "target": "name", "match_mode": "regex", "pattern": "凤凰|CGTN" }
    ]
  }
]
\`\`\`

Note: In the example above, since the channel names are in Chinese, the group names are also in Chinese. Always follow this principle.

## Now generate grouping rules for the following channel list:

${channelList}
`
  },
}

// ---------------------------------------------------------------------------
// Registry – add new templates here
// ---------------------------------------------------------------------------

const templates = [ALIAS_RULES_TEMPLATE, FILTER_RULES_TEMPLATE, GROUP_RULES_TEMPLATE]

/**
 * Get all registered prompt templates.
 * @returns {Array} template objects
 */
export function getPromptTemplates() {
  return templates
}

/**
 * Get a template by id.
 * @param {string} id
 * @returns {object|undefined}
 */
export function getPromptTemplate(id) {
  return templates.find((t) => t.id === id)
}

/**
 * Validate AI-returned JSON against the group rules schema.
 * Returns { valid: boolean, data?: Array, error?: string }.
 *
 * Expected shape:
 *   [ { group_name: string, rules: [ { target, match_mode, pattern } ] } ]
 */
export function validateGroupRulesJSON(jsonString) {
  const parsed = parseJSONSafe(jsonString)
  if (!parsed) return { valid: false, error: 'invalid_json' }

  if (!Array.isArray(parsed) || parsed.length === 0) {
    return { valid: false, error: 'not_array' }
  }

  for (let i = 0; i < parsed.length; i++) {
    const group = parsed[i]

    if (!group.group_name || typeof group.group_name !== 'string') {
      return { valid: false, error: 'missing_group_name', index: i }
    }

    if (!Array.isArray(group.rules) || group.rules.length === 0) {
      return { valid: false, error: 'missing_rules', index: i }
    }

    for (let j = 0; j < group.rules.length; j++) {
      const rule = group.rules[j]
      if (!rule.pattern || typeof rule.pattern !== 'string') {
        return { valid: false, error: 'missing_pattern', groupIndex: i, ruleIndex: j }
      }
      // Normalise: ensure target and match_mode have defaults
      if (!rule.target) rule.target = 'name'
      if (!rule.match_mode) rule.match_mode = 'regex'
    }
  }

  return { valid: true, data: parsed }
}

/**
 * Validate AI-returned JSON against the alias rules schema.
 * Returns { valid: boolean, data?: Array, error?: string }.
 *
 * Expected shape:
 *   [ { match_mode: string, pattern: string, replacement: string } ]
 */
export function validateAliasRulesJSON(jsonString) {
  const parsed = parseJSONSafe(jsonString)
  if (!parsed) return { valid: false, error: 'invalid_json' }

  if (!Array.isArray(parsed) || parsed.length === 0) {
    return { valid: false, error: 'not_array' }
  }

  for (let i = 0; i < parsed.length; i++) {
    const rule = parsed[i]

    if (!rule.pattern || typeof rule.pattern !== 'string') {
      return { valid: false, error: 'missing_pattern', index: i }
    }

    // replacement can be empty string (e.g. removing suffixes), so only check type
    if (rule.replacement === undefined || rule.replacement === null) {
      rule.replacement = ''
    }
    if (typeof rule.replacement !== 'string') {
      return { valid: false, error: 'invalid_replacement', index: i }
    }

    // Normalise match_mode
    if (!rule.match_mode) rule.match_mode = 'regex'
  }

  return { valid: true, data: parsed }
}

/**
 * Validate AI-returned JSON against the filter rules schema.
 * Returns { valid: boolean, data?: Array, error?: string }.
 *
 * Expected shape:
 *   [ { target: string, match_mode: string, pattern: string } ]
 */
export function validateFilterRulesJSON(jsonString) {
  const parsed = parseJSONSafe(jsonString)
  if (!parsed) return { valid: false, error: 'invalid_json' }

  if (!Array.isArray(parsed) || parsed.length === 0) {
    return { valid: false, error: 'not_array' }
  }

  for (let i = 0; i < parsed.length; i++) {
    const rule = parsed[i]

    if (!rule.pattern || typeof rule.pattern !== 'string') {
      return { valid: false, error: 'missing_pattern', index: i }
    }

    // Normalise defaults
    if (!rule.target) rule.target = 'name'
    if (!rule.match_mode) rule.match_mode = 'regex'
  }

  return { valid: true, data: parsed }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Parse a JSON string safely, stripping markdown code fences if present.
 * Returns the parsed value or null on failure.
 */
function parseJSONSafe(jsonString) {
  let cleaned = jsonString.trim()

  // Strip markdown code fences (```json ... ``` or ``` ... ```)
  const fenceMatch = cleaned.match(/^```(?:json)?\s*\n?([\s\S]*?)\n?\s*```$/)
  if (fenceMatch) {
    cleaned = fenceMatch[1].trim()
  }

  try {
    return JSON.parse(cleaned)
  } catch {
    return null
  }
}
