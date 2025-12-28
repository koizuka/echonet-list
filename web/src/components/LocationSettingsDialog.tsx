import { useState } from 'react';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { isJapanese } from '@/libs/languageHelper';
import { getLocationDisplayName, LOCATION_SEPARATOR, isSeparator } from '@/libs/locationHelper';
import type { LocationSettings, Device, PropertyDescriptionData } from '@/hooks/types';
import { Trash2, Plus, RotateCcw, GripVertical, Minus } from 'lucide-react';

type DialogMessages = {
  title: string;
  aliasSection: string;
  aliasName: string;
  selectLocation: string;
  addAlias: string;
  noAliases: string;
  noLocations: string;
  orderSection: string;
  noOrder: string;
  initializeOrder: string;
  resetOrder: string;
  addSeparator: string;
  separator: string;
  close: string;
  cancel: string;
  apply: string;
  aliasError: string;
  aliasErrorTooLong: string;
  aliasErrorMultipleHash: string;
  aliasErrorProhibitedChars: string;
  aliasWarningTruncated: string;
};

// Validation constants (must match backend)
const MAX_LOCATION_ALIAS_LENGTH = 32;
const PROHIBITED_CHARS_PATTERN = /[\t\n\r "!$%&'()*,/;<=>?@[\\\]^`{|}~]/;

function validateLocationAlias(alias: string, texts: DialogMessages): { valid: boolean; error?: string } {
  if (!alias.startsWith('#')) {
    return { valid: false, error: texts.aliasError };
  }
  if (alias.length <= 1) {
    return { valid: false, error: texts.aliasError };
  }
  if (alias.length > MAX_LOCATION_ALIAS_LENGTH) {
    return { valid: false, error: texts.aliasErrorTooLong };
  }
  if (alias.slice(1).includes('#')) {
    return { valid: false, error: texts.aliasErrorMultipleHash };
  }
  if (PROHIBITED_CHARS_PATTERN.test(alias.slice(1))) {
    return { valid: false, error: texts.aliasErrorProhibitedChars };
  }
  return { valid: true };
}

interface SortableItemProps {
  id: string;
  displayName: string;
  disabled: boolean;
}

function SortableItem({ id, displayName, disabled }: SortableItemProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id, disabled });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition: transition ?? 'transform 150ms ease, box-shadow 150ms ease',
  };

  // Enhanced draggable card with better visual feedback
  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`
        flex items-center gap-3 text-sm rounded-md px-3 py-2.5 select-none touch-none
        border border-transparent
        transition-all duration-150 ease-out
        ${isDragging
          ? 'bg-accent shadow-lg scale-[1.02] border-primary/20 z-10 relative'
          : 'bg-muted/50 hover:bg-muted hover:border-border'
        }
        ${disabled
          ? 'opacity-50 cursor-not-allowed'
          : 'cursor-grab active:cursor-grabbing'
        }
      `}
      {...attributes}
      {...listeners}
    >
      <GripVertical className={`h-4 w-4 flex-shrink-0 transition-colors ${
        isDragging ? 'text-primary' : 'text-muted-foreground/60'
      }`} />
      <span className="flex-1 truncate font-medium">{displayName}</span>
    </div>
  );
}

interface SortableSeparatorProps {
  id: string;
  displayName: string;
  disabled: boolean;
  onDelete: () => void;
}

function SortableSeparator({ id, displayName, disabled, onDelete }: SortableSeparatorProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id, disabled });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition: transition ?? 'transform 150ms ease, box-shadow 150ms ease',
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`
        flex items-center gap-2 text-sm rounded-md px-3 py-1.5 select-none touch-none
        transition-all duration-150 ease-out
        ${isDragging
          ? 'bg-primary/5 shadow-md scale-[1.01] z-10 relative'
          : 'hover:bg-muted/30'
        }
        ${disabled
          ? 'opacity-50 cursor-not-allowed'
          : 'cursor-grab active:cursor-grabbing'
        }
      `}
    >
      <div
        className="flex items-center gap-2 flex-1 min-w-0"
        {...attributes}
        {...listeners}
      >
        <GripVertical className={`h-3.5 w-3.5 flex-shrink-0 transition-colors ${
          isDragging ? 'text-primary' : 'text-muted-foreground/30'
        }`} />
        <div className="flex-1 flex items-center gap-2">
          <div className="flex-1 h-px bg-border" />
          <span className="text-[10px] font-display text-muted-foreground/70 uppercase tracking-wider whitespace-nowrap">
            {displayName}
          </span>
          <div className="flex-1 h-px bg-border" />
        </div>
      </div>
      <button
        type="button"
        className="h-5 w-5 p-0 flex items-center justify-center rounded-sm text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-colors"
        onClick={onDelete}
        disabled={disabled}
      >
        <Trash2 className="h-3 w-3" />
      </button>
    </div>
  );
}

interface LocationSettingsDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  locationSettings: LocationSettings;
  availableLocations: string[]; // List of location values from devices
  devices: Record<string, Device>;
  propertyDescriptions: Record<string, PropertyDescriptionData>;
  onAddLocationAlias: (alias: string, value: string) => Promise<unknown>;
  onDeleteLocationAlias: (alias: string) => Promise<unknown>;
  onSetLocationOrder: (order: string[]) => Promise<unknown>;
  isConnected: boolean;
}

export function LocationSettingsDialog({
  isOpen,
  onOpenChange,
  locationSettings,
  availableLocations,
  devices,
  propertyDescriptions,
  onAddLocationAlias,
  onDeleteLocationAlias,
  onSetLocationOrder,
  isConnected,
}: LocationSettingsDialogProps) {
  const [newAliasName, setNewAliasName] = useState('');
  const [newAliasValue, setNewAliasValue] = useState('');
  const [aliasError, setAliasError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  // Phase 3: Pending order for OK/Cancel pattern
  const [pendingOrder, setPendingOrder] = useState<string[] | null>(null);
  const hasOrderChanges = pendingOrder !== null;
  // Use pending order if available, otherwise use current settings
  const displayOrder = pendingOrder ?? locationSettings.order;

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const messages: Record<'en' | 'ja', DialogMessages> = {
    en: {
      title: 'Location Settings',
      aliasSection: 'Location Aliases',
      aliasName: 'Alias (e.g. #2F Bedroom)',
      selectLocation: 'Select location',
      addAlias: 'Add',
      noAliases: 'No aliases defined',
      noLocations: 'No locations available',
      orderSection: 'Display Order',
      noOrder: 'Using default order',
      initializeOrder: 'Customize Order',
      resetOrder: 'Reset Order',
      addSeparator: 'Add Separator',
      separator: 'Separator',
      close: 'Close',
      cancel: 'Cancel',
      apply: 'Apply',
      aliasError: 'Alias must start with #',
      aliasErrorTooLong: `Alias must be ${MAX_LOCATION_ALIAS_LENGTH} characters or less`,
      aliasErrorMultipleHash: 'Alias cannot contain # after the first character',
      aliasErrorProhibitedChars: 'Alias contains prohibited characters',
      aliasWarningTruncated: `Input truncated to ${MAX_LOCATION_ALIAS_LENGTH} characters`,
    },
    ja: {
      title: '設置場所設定',
      aliasSection: 'エイリアス',
      aliasName: 'エイリアス (例: #2F寝室)',
      selectLocation: '設置場所を選択',
      addAlias: '追加',
      noAliases: 'エイリアスが設定されていません',
      noLocations: '設置場所がありません',
      orderSection: '表示順',
      noOrder: 'デフォルト順',
      initializeOrder: '順序をカスタマイズ',
      resetOrder: '順序リセット',
      addSeparator: 'セパレータを追加',
      separator: 'セパレータ',
      close: '閉じる',
      cancel: 'キャンセル',
      apply: '適用',
      aliasError: 'エイリアスは # で始まる必要があります',
      aliasErrorTooLong: `エイリアスは${MAX_LOCATION_ALIAS_LENGTH}文字以内で入力してください`,
      aliasErrorMultipleHash: 'エイリアスの2文字目以降に#は使用できません',
      aliasErrorProhibitedChars: 'エイリアスに使用できない文字が含まれています',
      aliasWarningTruncated: `入力が${MAX_LOCATION_ALIAS_LENGTH}文字に切り詰められました`,
    },
  };

  const texts = isJapanese() ? messages.ja : messages.en;

  // Helper to get translated location name (without alias lookup - for alias section)
  const getTranslatedLocationName = (locationId: string): string => {
    // Don't pass locationSettings to avoid alias lookup - we want the translated raw name
    return getLocationDisplayName(locationId, devices, propertyDescriptions, undefined);
  };

  // Helper to get display name with alias support (for order section)
  const getDisplayNameWithAlias = (locationId: string): string => {
    // Use locationSettings to show alias if available
    return getLocationDisplayName(locationId, devices, propertyDescriptions, locationSettings);
  };

  const handleAddAlias = async () => {
    if (!newAliasName || !newAliasValue) {
      return;
    }

    const validation = validateLocationAlias(newAliasName, texts);
    if (!validation.valid) {
      setAliasError(validation.error ?? '');
      return;
    }

    setIsLoading(true);
    setAliasError('');
    try {
      await onAddLocationAlias(newAliasName, newAliasValue);
      setNewAliasName('');
      setNewAliasValue('');
    } finally {
      setIsLoading(false);
    }
  };

  // Auto-insert # prefix when user types without it
  const handleAliasNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let value = e.target.value;
    // Auto-insert # if the first character is not #
    if (value.length > 0 && !value.startsWith('#')) {
      value = '#' + value;
    }
    // Trim to max length (handles paste of long strings)
    let wasTruncated = false;
    if (value.length > MAX_LOCATION_ALIAS_LENGTH) {
      value = value.slice(0, MAX_LOCATION_ALIAS_LENGTH);
      wasTruncated = true;
    }
    setNewAliasName(value);
    // Show warning if truncated, otherwise clear error
    setAliasError(wasTruncated ? texts.aliasWarningTruncated : '');
  };

  // Clear state when dialog closes
  const handleOpenChange = (open: boolean) => {
    if (!open) {
      setPendingOrder(null);
      setNewAliasName('');
      setNewAliasValue('');
      setAliasError('');
    }
    onOpenChange(open);
  };

  const handleDeleteAlias = async (alias: string) => {
    setIsLoading(true);
    try {
      await onDeleteLocationAlias(alias);
    } finally {
      setIsLoading(false);
    }
  };

  const handleResetOrder = async () => {
    setIsLoading(true);
    try {
      await onSetLocationOrder([]);
      setPendingOrder(null);
    } finally {
      setIsLoading(false);
    }
  };

  const handleInitializeOrder = async () => {
    setIsLoading(true);
    try {
      // Initialize with all available locations sorted
      const newOrder = [...availableLocations].sort();
      await onSetLocationOrder(newOrder);
      setPendingOrder(null);
    } finally {
      setIsLoading(false);
    }
  };

  // Phase 3: Apply pending order changes
  const handleApplyOrderChanges = async () => {
    if (pendingOrder) {
      setIsLoading(true);
      try {
        await onSetLocationOrder(pendingOrder);
        setPendingOrder(null);
      } finally {
        setIsLoading(false);
      }
    }
  };

  // Phase 3: Cancel pending order changes
  const handleCancelOrderChanges = () => {
    setPendingOrder(null);
  };

  // Convert order array to unique IDs for SortableContext
  // Separators get unique IDs based on array index (e.g., "---:0", "---:3") for deterministic generation
  const orderToUniqueIds = (order: string[]): string[] => {
    return order.map((item, index) => {
      if (isSeparator(item)) {
        return `${LOCATION_SEPARATOR}:${index}`;
      }
      return item;
    });
  };

  // Convert unique IDs back to order array for saving
  const uniqueIdsToOrder = (ids: string[]): string[] => {
    return ids.map(id => {
      if (id.startsWith(`${LOCATION_SEPARATOR}:`)) {
        return LOCATION_SEPARATOR;
      }
      return id;
    });
  };

  // Check if a unique ID is a separator
  const isUniqueIdSeparator = (id: string): boolean => {
    return id.startsWith(`${LOCATION_SEPARATOR}:`);
  };

  const displayOrderWithUniqueIds = orderToUniqueIds(displayOrder);

  // Add separator to the end of the order list
  const handleAddSeparator = () => {
    const currentOrder = pendingOrder ?? locationSettings.order;
    const newOrder = [...currentOrder, LOCATION_SEPARATOR];
    setPendingOrder(newOrder);
  };

  // Delete separator at specific index
  const handleDeleteSeparator = (uniqueId: string) => {
    const currentIds = displayOrderWithUniqueIds;
    const index = currentIds.indexOf(uniqueId);
    if (index !== -1) {
      const newIds = currentIds.filter((_, i) => i !== index);
      const newOrder = uniqueIdsToOrder(newIds);
      setPendingOrder(newOrder);
    }
  };

  // Updated drag handler to work with unique IDs
  const handleDragEndWithUniqueIds = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      const currentIds = displayOrderWithUniqueIds;
      const oldIndex = currentIds.indexOf(active.id as string);
      const newIndex = currentIds.indexOf(over.id as string);
      const newIds = arrayMove(currentIds, oldIndex, newIndex);
      const newOrder = uniqueIdsToOrder(newIds);
      setPendingOrder(newOrder);
    }
  };

  const aliasEntries = Object.entries(locationSettings.aliases);

  return (
    <AlertDialog open={isOpen} onOpenChange={handleOpenChange}>
      <AlertDialogContent className="max-w-lg max-h-[80vh] flex flex-col gap-0 p-0 overflow-hidden">
        {/* Header with subtle background */}
        <AlertDialogHeader className="flex-shrink-0 px-6 py-4 border-b bg-muted/30">
          <AlertDialogTitle className="text-lg font-semibold tracking-tight">
            {texts.title}
          </AlertDialogTitle>
          <AlertDialogDescription className="sr-only">
            {texts.title}
          </AlertDialogDescription>
        </AlertDialogHeader>

        {/* Scrollable content area */}
        <div className="flex-1 overflow-y-auto min-h-0 px-6 py-5 space-y-6">
          {/* Alias Section */}
          <section className="space-y-3">
            <h3 className="text-sm font-semibold text-foreground/80 uppercase tracking-wider">
              {texts.aliasSection}
            </h3>

            {/* Alias List */}
            <div className="space-y-2">
              {aliasEntries.length === 0 ? (
                <p className="text-sm text-muted-foreground py-2">{texts.noAliases}</p>
              ) : (
                aliasEntries.map(([alias, value]) => (
                  <div
                    key={alias}
                    className="flex items-center gap-3 text-sm bg-muted/40 rounded-md px-3 py-2 group hover:bg-muted/60 transition-colors"
                  >
                    <span className="font-medium text-primary/90 flex-1 truncate">{alias}</span>
                    <span className="text-muted-foreground/60 text-xs">→</span>
                    <span className="text-muted-foreground flex-1 truncate">{getTranslatedLocationName(value)}</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-7 w-7 p-0 opacity-60 hover:opacity-100 hover:bg-destructive/10 hover:text-destructive transition-all"
                      onClick={() => handleDeleteAlias(alias)}
                      disabled={isLoading || !isConnected}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                ))
              )}
            </div>

            {/* Add Alias Form */}
            <div className="flex gap-2 pt-1">
              <Input
                placeholder={texts.aliasName}
                value={newAliasName}
                onChange={handleAliasNameChange}
                className="flex-1 h-9 text-sm"
                disabled={isLoading || !isConnected}
                maxLength={MAX_LOCATION_ALIAS_LENGTH}
              />
              <Select
                value={newAliasValue}
                onValueChange={setNewAliasValue}
                disabled={isLoading || !isConnected || availableLocations.length === 0}
              >
                <SelectTrigger className="flex-1 h-9 text-sm">
                  <SelectValue placeholder={texts.selectLocation} />
                </SelectTrigger>
                <SelectContent>
                  {availableLocations.length === 0 ? (
                    <SelectItem value="_no_locations_" disabled>
                      {texts.noLocations}
                    </SelectItem>
                  ) : (
                    availableLocations.map((location) => (
                      <SelectItem key={location} value={location}>
                        {getTranslatedLocationName(location)}
                      </SelectItem>
                    ))
                  )}
                </SelectContent>
              </Select>
              <Button
                variant="outline"
                size="sm"
                onClick={handleAddAlias}
                disabled={isLoading || !isConnected || !newAliasName || !newAliasValue}
                className="h-9 px-3"
              >
                <Plus className="h-4 w-4 mr-1.5" />
                {texts.addAlias}
              </Button>
            </div>
            {aliasError && (
              <p className="text-xs text-destructive font-medium mt-1.5 flex items-center gap-1">
                <span className="inline-block w-1 h-1 rounded-full bg-destructive" />
                {aliasError}
              </p>
            )}
          </section>

          {/* Divider */}
          <div className="border-t border-border/50" />

          {/* Order Section */}
          <section className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-foreground/80 uppercase tracking-wider">
                {texts.orderSection}
              </h3>
              <div className="flex items-center gap-2">
                {displayOrder.length > 0 && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleAddSeparator}
                    disabled={isLoading || !isConnected}
                    className="h-7 text-xs text-muted-foreground hover:text-foreground"
                  >
                    <Minus className="h-3 w-3 mr-1.5" />
                    {texts.addSeparator}
                  </Button>
                )}
                {locationSettings.order.length > 0 && !hasOrderChanges && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleResetOrder}
                    disabled={isLoading || !isConnected}
                    className="h-7 text-xs text-muted-foreground hover:text-foreground"
                  >
                    <RotateCcw className="h-3 w-3 mr-1.5" />
                    {texts.resetOrder}
                  </Button>
                )}
              </div>
            </div>

            <div className="space-y-1.5">
              {displayOrder.length === 0 ? (
                <div className="space-y-3 py-2">
                  <p className="text-sm text-muted-foreground">{texts.noOrder}</p>
                  {availableLocations.length > 0 && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleInitializeOrder}
                      disabled={isLoading || !isConnected}
                      className="h-8 text-xs"
                    >
                      <Plus className="h-3.5 w-3.5 mr-1.5" />
                      {texts.initializeOrder}
                    </Button>
                  )}
                </div>
              ) : (
                <DndContext
                  sensors={sensors}
                  collisionDetection={closestCenter}
                  onDragEnd={handleDragEndWithUniqueIds}
                >
                  <SortableContext
                    items={displayOrderWithUniqueIds}
                    strategy={verticalListSortingStrategy}
                  >
                    {displayOrderWithUniqueIds.map((uniqueId) => (
                      isUniqueIdSeparator(uniqueId) ? (
                        <SortableSeparator
                          key={uniqueId}
                          id={uniqueId}
                          displayName={texts.separator}
                          disabled={isLoading || !isConnected}
                          onDelete={() => handleDeleteSeparator(uniqueId)}
                        />
                      ) : (
                        <SortableItem
                          key={uniqueId}
                          id={uniqueId}
                          displayName={getDisplayNameWithAlias(uniqueId)}
                          disabled={isLoading || !isConnected}
                        />
                      )
                    ))}
                  </SortableContext>
                </DndContext>
              )}
            </div>
          </section>
        </div>

        {/* Fixed footer */}
        <AlertDialogFooter className="flex-shrink-0 border-t bg-muted/20 px-6 py-4 gap-2">
          {hasOrderChanges && (
            <>
              <Button
                variant="ghost"
                onClick={handleCancelOrderChanges}
                disabled={isLoading}
                className="text-muted-foreground hover:text-foreground"
              >
                {texts.cancel}
              </Button>
              <Button
                onClick={handleApplyOrderChanges}
                disabled={isLoading}
                className="min-w-[80px]"
              >
                {texts.apply}
              </Button>
            </>
          )}
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={isLoading && hasOrderChanges}
          >
            {texts.close}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
