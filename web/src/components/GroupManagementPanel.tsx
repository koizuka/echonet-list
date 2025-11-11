import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Edit2, Users, Trash2, MoreVertical } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { GroupBulkControl } from './GroupBulkControl';
import { getCurrentLocale } from '@/libs/languageHelper';
import type { Device, PropertyValue } from '../hooks/types';
import type { LogEntry } from '../hooks/useLogNotifications';

// Localized message templates for group management
const GROUP_MANAGEMENT_MESSAGES = {
  menu: {
    settings: {
      en: 'Group settings',
      ja: 'グループ設定'
    },
    open_settings: {
      en: 'Open group settings menu',
      ja: 'グループ設定メニューを開く'
    },
    rename: {
      en: 'Rename group',
      ja: 'グループ名を変更'
    },
    edit_members: {
      en: 'Edit members',
      ja: 'メンバーを編集'
    },
    delete_group: {
      en: 'Delete group',
      ja: 'グループを削除'
    },
    done_editing: {
      en: 'Done editing',
      ja: '編集を終了'
    },
    stop_editing_members: {
      en: 'Stop editing members',
      ja: 'メンバー編集を終了'
    }
  },
  dialog: {
    delete_confirmation: {
      en: 'Delete group confirmation',
      ja: 'グループの削除確認'
    },
    delete_message: {
      en: 'Are you sure you want to delete {groupName}?',
      ja: '{groupName} を削除してもよろしいですか？'
    },
    cannot_undo: {
      en: 'This action cannot be undone.',
      ja: 'この操作は取り消せません。'
    },
    cancel: {
      en: 'Cancel',
      ja: 'キャンセル'
    },
    delete: {
      en: 'Delete',
      ja: '削除する'
    }
  }
} as const;

interface GroupManagementPanelProps {
  groupName: string;
  onRename: () => void;
  onDelete: () => void;
  onEditMembers: () => void;
  isEditingMembers?: boolean;
  onDoneEditingMembers?: () => void;
  isConnected?: boolean;
  devices?: Device[];
  onPropertyChange?: (target: string, epc: string, value: PropertyValue) => Promise<void>;
  addLogEntry?: (log: LogEntry) => void;
}

export function GroupManagementPanel({
  groupName,
  onRename,
  onDelete,
  onEditMembers,
  isEditingMembers = false,
  onDoneEditingMembers,
  isConnected = true,
  devices = [],
  onPropertyChange,
  addLogEntry,
}: GroupManagementPanelProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const locale = getCurrentLocale();

  const handleDeleteClick = () => {
    setShowDeleteConfirm(true);
  };

  const handleConfirmDelete = () => {
    onDelete();
    setShowDeleteConfirm(false);
  };

  const handleCancelDelete = () => {
    setShowDeleteConfirm(false);
  };

  return (
    <div className="space-y-4">
      {/* Bulk Power Control + Group Management Controls */}
      {!isEditingMembers && (
        <div className="flex items-center gap-2 mb-4">
          {/* Bulk Power Control - Left side */}
          {devices.length > 0 && onPropertyChange && (
            <GroupBulkControl
              devices={devices}
              onPropertyChange={onPropertyChange}
              addLogEntry={addLogEntry}
            />
          )}

          {/* Group Settings Menu - Right side */}
          <div className="ml-auto">
            <DropdownMenu onOpenChange={setIsMenuOpen}>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!isConnected}
                  title={GROUP_MANAGEMENT_MESSAGES.menu.settings[locale]}
                  aria-label={GROUP_MANAGEMENT_MESSAGES.menu.open_settings[locale]}
                  aria-haspopup="true"
                  aria-expanded={isMenuOpen}
                >
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={onRename} disabled={!isConnected}>
                  <Edit2 className="h-4 w-4 mr-2" />
                  {GROUP_MANAGEMENT_MESSAGES.menu.rename[locale]}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={onEditMembers} disabled={!isConnected}>
                  <Users className="h-4 w-4 mr-2" />
                  {GROUP_MANAGEMENT_MESSAGES.menu.edit_members[locale]}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={handleDeleteClick}
                  disabled={!isConnected}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  {GROUP_MANAGEMENT_MESSAGES.menu.delete_group[locale]}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      )}

      {/* Member Editing Mode */}
      {isEditingMembers && (
        <div className="mb-4">
          <Button
            variant="outline"
            size="sm"
            onClick={onDoneEditingMembers}
            title={GROUP_MANAGEMENT_MESSAGES.menu.stop_editing_members[locale]}
          >
            <Users className="h-4 w-4 sm:mr-2" />
            <span className="hidden sm:inline">{GROUP_MANAGEMENT_MESSAGES.menu.done_editing[locale]}</span>
          </Button>
        </div>
      )}

      {/* Delete Confirmation Dialog */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <button 
            className="fixed inset-0 bg-black/50 cursor-default" 
            onClick={handleCancelDelete}
            aria-label="Close dialog"
          />
          <Card className="relative z-10 max-w-md mx-4">
            <CardContent className="space-y-4">
              <h3 className="text-lg font-semibold">{GROUP_MANAGEMENT_MESSAGES.dialog.delete_confirmation[locale]}</h3>
              <p className="text-sm">
                {GROUP_MANAGEMENT_MESSAGES.dialog.delete_message[locale].replace('{groupName}', groupName)}
              </p>
              <p className="text-sm text-muted-foreground">
                {GROUP_MANAGEMENT_MESSAGES.dialog.cannot_undo[locale]}
              </p>
              <div className="flex gap-2 justify-end">
                <Button
                  variant="outline"
                  onClick={handleCancelDelete}
                >
                  {GROUP_MANAGEMENT_MESSAGES.dialog.cancel[locale]}
                </Button>
                <Button
                  variant="destructive"
                  onClick={handleConfirmDelete}
                >
                  {GROUP_MANAGEMENT_MESSAGES.dialog.delete[locale]}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}