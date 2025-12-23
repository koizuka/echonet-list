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
import { getLocationDisplayName } from '@/libs/locationHelper';
import type { LocationSettings, Device, PropertyDescriptionData } from '@/hooks/types';
import { Trash2, Plus, RotateCcw, GripVertical } from 'lucide-react';

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
  close: string;
  aliasError: string;
};

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
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="flex items-center gap-2 text-sm bg-background rounded px-1 py-0.5"
    >
      <button
        {...attributes}
        {...listeners}
        className={`touch-none p-1 ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-grab active:cursor-grabbing'}`}
        disabled={disabled}
      >
        <GripVertical className="h-3 w-3 text-muted-foreground" />
      </button>
      <span className="font-mono flex-1 truncate">{displayName}</span>
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
      close: 'Close',
      aliasError: 'Alias must start with #',
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
      close: '閉じる',
      aliasError: 'エイリアスは # で始まる必要があります',
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
    if (!newAliasName.startsWith('#')) {
      setAliasError(texts.aliasError);
      return;
    }
    if (!newAliasName || !newAliasValue) {
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
    } finally {
      setIsLoading(false);
    }
  };

  const handleInitializeOrder = async () => {
    setIsLoading(true);
    try {
      // Initialize with all available locations sorted
      await onSetLocationOrder([...availableLocations].sort());
    } finally {
      setIsLoading(false);
    }
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      const oldIndex = locationSettings.order.indexOf(active.id as string);
      const newIndex = locationSettings.order.indexOf(over.id as string);
      const newOrder = arrayMove(locationSettings.order, oldIndex, newIndex);

      setIsLoading(true);
      try {
        await onSetLocationOrder(newOrder);
      } finally {
        setIsLoading(false);
      }
    }
  };

  const aliasEntries = Object.entries(locationSettings.aliases);

  return (
    <AlertDialog open={isOpen} onOpenChange={onOpenChange}>
      <AlertDialogContent className="max-w-lg max-h-[80vh] overflow-y-auto">
        <AlertDialogHeader>
          <AlertDialogTitle>{texts.title}</AlertDialogTitle>
          <AlertDialogDescription className="sr-only">
            {texts.title}
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className="space-y-6">
          {/* Alias Section */}
          <div>
            <h3 className="text-sm font-medium mb-2">{texts.aliasSection}</h3>

            {/* Alias List */}
            <div className="space-y-2 mb-3">
              {aliasEntries.length === 0 ? (
                <p className="text-sm text-muted-foreground">{texts.noAliases}</p>
              ) : (
                aliasEntries.map(([alias, value]) => (
                  <div key={alias} className="flex items-center gap-2 text-sm">
                    <span className="font-mono flex-1 truncate">{alias}</span>
                    <span className="text-muted-foreground">→</span>
                    <span className="font-mono flex-1 truncate">{getTranslatedLocationName(value)}</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 w-6 p-0"
                      onClick={() => handleDeleteAlias(alias)}
                      disabled={isLoading || !isConnected}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                ))
              )}
            </div>

            {/* Add Alias Form */}
            <div className="flex gap-2">
              <Input
                placeholder={texts.aliasName}
                value={newAliasName}
                onChange={(e) => {
                  setNewAliasName(e.target.value);
                  setAliasError('');
                }}
                className="flex-1 h-8 text-sm"
                disabled={isLoading || !isConnected}
              />
              <Select
                value={newAliasValue}
                onValueChange={setNewAliasValue}
                disabled={isLoading || !isConnected || availableLocations.length === 0}
              >
                <SelectTrigger className="flex-1 h-8 text-sm">
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
                className="h-8"
              >
                <Plus className="h-3 w-3 mr-1" />
                {texts.addAlias}
              </Button>
            </div>
            {aliasError && (
              <p className="text-xs text-destructive mt-1">{aliasError}</p>
            )}
          </div>

          {/* Order Section */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <h3 className="text-sm font-medium">{texts.orderSection}</h3>
              {locationSettings.order.length > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleResetOrder}
                  disabled={isLoading || !isConnected}
                  className="h-6 text-xs"
                >
                  <RotateCcw className="h-3 w-3 mr-1" />
                  {texts.resetOrder}
                </Button>
              )}
            </div>

            <div className="space-y-1">
              {locationSettings.order.length === 0 ? (
                <div className="space-y-2">
                  <p className="text-sm text-muted-foreground">{texts.noOrder}</p>
                  {availableLocations.length > 0 && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleInitializeOrder}
                      disabled={isLoading || !isConnected}
                      className="h-7 text-xs"
                    >
                      <Plus className="h-3 w-3 mr-1" />
                      {texts.initializeOrder}
                    </Button>
                  )}
                </div>
              ) : (
                <DndContext
                  sensors={sensors}
                  collisionDetection={closestCenter}
                  onDragEnd={handleDragEnd}
                >
                  <SortableContext
                    items={locationSettings.order}
                    strategy={verticalListSortingStrategy}
                  >
                    {locationSettings.order.map((item) => (
                      <SortableItem
                        key={item}
                        id={item}
                        displayName={getDisplayNameWithAlias(item)}
                        disabled={isLoading || !isConnected}
                      />
                    ))}
                  </SortableContext>
                </DndContext>
              )}
            </div>
          </div>
        </div>

        <AlertDialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {texts.close}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
