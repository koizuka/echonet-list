export function validateGroupName(name: string, existingGroups?: string[]): string | undefined {
  // Check if name starts with @
  if (!name.startsWith('@')) {
    return 'グループ名は @ で始まる必要があります';
  }

  // Check if name has at least 1 character after @
  if (name.length <= 1) {
    return 'グループ名は @ の後に少なくとも1文字必要です';
  }

  // Check if name contains whitespace
  if (/\s/.test(name)) {
    return 'グループ名に空白文字を含めることはできません';
  }

  // Check for duplicate names
  if (existingGroups && existingGroups.includes(name)) {
    return 'このグループ名は既に使用されています';
  }

  return undefined;
}