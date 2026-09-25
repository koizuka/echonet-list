import { getCurrentLocale } from './languageHelper';

const messages = {
  en: {
    empty: 'Please enter an alias name',
    hex: 'Names that read as an even-length hex string are not allowed',
    symbol: 'Names cannot start with a symbol',
  },
  ja: {
    empty: 'エイリアス名を入力してください',
    hex: '16進数として読める偶数桁の名前は使用できません',
    symbol: '記号で始まる名前は使用できません',
  },
};

/**
 * Validates a device alias according to the same rules as the server.
 * @param alias The alias to validate
 * @returns Error message if invalid, undefined if valid
 */
export function validateDeviceAlias(alias: string): string | undefined {
  const texts = messages[getCurrentLocale()];

  // Empty string is not allowed
  if (alias === '') {
    return texts.empty;
  }

  // Check if it's an even-length hex string
  const hexPattern = /^[0-9A-Fa-f]+$/;
  if (alias.length % 2 === 0 && alias.length > 0 && hexPattern.test(alias)) {
    return texts.hex;
  }

  // Check if it starts with a symbol
  const invalidFirstCharPattern = /^[!"#$%&'()*+,./:;<=>?@[\\\]^_{|}~-]/;
  if (invalidFirstCharPattern.test(alias)) {
    return texts.symbol;
  }

  // Check for emojis at the start (they are considered symbols)
  // This is a simplified check - emojis are in various Unicode ranges
  const emojiPattern = /^[\u{1F300}-\u{1F9FF}\u{1F600}-\u{1F64F}\u{1F680}-\u{1F6FF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}]/u;
  if (emojiPattern.test(alias)) {
    return texts.symbol;
  }

  return undefined;
}