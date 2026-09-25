import { getCurrentLocale } from './languageHelper';

const messages = {
  en: {
    noPrefix: 'Group names must start with @',
    tooShort: 'Group names need at least one character after @',
    whitespace: 'Group names cannot contain whitespace',
    duplicate: 'This group name is already in use',
  },
  ja: {
    noPrefix: 'グループ名は @ で始まる必要があります',
    tooShort: 'グループ名は @ の後に少なくとも1文字必要です',
    whitespace: 'グループ名に空白文字を含めることはできません',
    duplicate: 'このグループ名は既に使用されています',
  },
};

export function validateGroupName(name: string, existingGroups?: string[]): string | undefined {
  const texts = messages[getCurrentLocale()];

  // Check if name starts with @
  if (!name.startsWith('@')) {
    return texts.noPrefix;
  }

  // Check if name has at least 1 character after @
  if (name.length <= 1) {
    return texts.tooShort;
  }

  // Check if name contains whitespace
  if (/\s/.test(name)) {
    return texts.whitespace;
  }

  // Check for duplicate names
  if (existingGroups && existingGroups.includes(name)) {
    return texts.duplicate;
  }

  return undefined;
}

/**
 * Returns `baseName`, or `baseName-1`, `baseName-2`, ... if it is already used.
 */
export function getUnusedGroupName(baseName: string, usedNames: Iterable<string>): string {
  const used = new Set(usedNames);
  let candidate = baseName;
  for (let suffix = 1; used.has(candidate); suffix++) {
    candidate = `${baseName}-${suffix}`;
  }
  return candidate;
}
