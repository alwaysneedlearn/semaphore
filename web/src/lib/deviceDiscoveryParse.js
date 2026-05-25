export function stripAnsiCodes(input) {
  if (!input) {
    return '';
  }
  const ESC = '\u001b';
  let i = 0;
  let out = '';
  while (i < input.length) {
    if (input[i] === ESC && input[i + 1] === '[') {
      i += 2;
      while (i < input.length) {
        const ch = input[i];
        if ((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')) {
          i += 1;
          break;
        }
        i += 1;
      }
    } else {
      out += input[i];
      i += 1;
    }
  }
  return out;
}

function tryParseJsonArray(candidate) {
  if (!candidate) {
    return null;
  }
  try {
    const parsed = JSON.parse(candidate.trim());
    return Array.isArray(parsed) ? parsed : null;
  } catch (e) {
    return null;
  }
}

function extractPrefixedDiscoveryJson(text) {
  if (!text) {
    return null;
  }
  const marker = 'SEMAPHORE_DISCOVERY_JSON=';
  const idx = text.lastIndexOf(marker);
  if (idx < 0) {
    return null;
  }
  const after = text.slice(idx + marker.length).trim();
  const endLine = after.indexOf('\n');
  const candidate = (endLine >= 0 ? after.slice(0, endLine) : after).trim();
  return tryParseJsonArray(candidate);
}

function tryParseYamlLikeDiscoveryList(text) {
  if (!text || !text.includes('device_status:')) {
    return null;
  }

  const lines = text.split('\n');
  const items = [];
  let current = null;

  const parseValue = (v) => {
    const raw = (v || '').trim();
    if (!raw) {
      return '';
    }
    if ((raw.startsWith('"') && raw.endsWith('"')) || (raw.startsWith('\'') && raw.endsWith('\''))) {
      return raw.slice(1, -1);
    }
    return raw;
  };

  for (let i = 0; i < lines.length; i += 1) {
    const line = stripAnsiCodes(lines[i] || '');
    const itemStart = line.match(/^\s*-\s+([a-z_]+)\s*:\s*(.*)$/i);
    if (itemStart) {
      if (current) {
        items.push(current);
      }
      current = { [itemStart[1]]: parseValue(itemStart[2]) };
    } else {
      const kv = line.match(/^\s{2,}([a-z_]+)\s*:\s*(.*)$/i);
      if (kv && current) {
        current[kv[1]] = parseValue(kv[2]);
      }
    }
  }

  if (current) {
    items.push(current);
  }

  if (items.length === 0 || !items.some((x) => x.ip_address || x.hostname)) {
    return null;
  }

  return items;
}

/** Parse discovered device rows from Ansible task log text. */
export function extractJsonArrayFromText(text) {
  if (!text) {
    return null;
  }

  const clean = stripAnsiCodes(text);

  const prefixed = extractPrefixedDiscoveryJson(clean);
  if (prefixed) {
    return prefixed;
  }

  const direct = tryParseJsonArray(clean);
  if (direct) {
    return direct;
  }

  const escapedFieldRegex = /"(stdout|msg)"\s*:\s*"((?:\\.|[^"\\])*)"/g;
  let escapedFieldMatch = escapedFieldRegex.exec(clean);
  while (escapedFieldMatch !== null) {
    try {
      const unescaped = JSON.parse(`"${escapedFieldMatch[2]}"`);
      const prefixedField = extractPrefixedDiscoveryJson(unescaped);
      if (prefixedField) {
        return prefixedField;
      }
      const fromField = tryParseJsonArray(unescaped);
      if (fromField) {
        return fromField;
      }
    } catch (e) {
      // continue
    }
    escapedFieldMatch = escapedFieldRegex.exec(clean);
  }

  const arrayLikeMatches = clean.match(/\[\s*\{[\s\S]*?\}\s*\]/g) || [];
  for (let i = arrayLikeMatches.length - 1; i >= 0; i -= 1) {
    const parsed = tryParseJsonArray(arrayLikeMatches[i]);
    if (parsed) {
      return parsed;
    }
  }

  const yamlLike = tryParseYamlLikeDiscoveryList(clean);
  if (yamlLike) {
    return yamlLike;
  }

  return null;
}

export function normalizeDiscoveredRows(parsed) {
  return (parsed || [])
    .map((x) => ({
      hostname: (x.hostname || '').trim(),
      ip_address: (x.ip_address || x.ip || '').trim(),
      device_status: x.device_status || x.status || 'unhealthy',
      rdp_status: x.rdp_status || 'offline',
      winrm_status: x.winrm_status || 'offline',
      api_status: x.api_status || 'offline',
      abnormal_reason: x.abnormal_reason || null,
    }))
    .filter((x) => x.ip_address || x.hostname);
}
