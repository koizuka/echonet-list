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
import type { Device, PropertyValue } from '../hooks/types';

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
}: GroupManagementPanelProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

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
            />
          )}

          {/* Group Settings Menu - Right side */}
          <div className="ml-auto">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!isConnected}
                  title="グループ設定"
                >
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={onRename} disabled={!isConnected}>
                  <Edit2 className="h-4 w-4 mr-2" />
                  グループ名を変更
                </DropdownMenuItem>
                <DropdownMenuItem onClick={onEditMembers} disabled={!isConnected}>
                  <Users className="h-4 w-4 mr-2" />
                  メンバーを編集
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={handleDeleteClick}
                  disabled={!isConnected}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  グループを削除
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
            title="メンバー編集を終了"
          >
            <Users className="h-4 w-4 sm:mr-2" />
            <span className="hidden sm:inline">編集を終了</span>
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
              <h3 className="text-lg font-semibold">グループの削除確認</h3>
              <p className="text-sm">
                {groupName} を削除してもよろしいですか？
              </p>
              <p className="text-sm text-muted-foreground">
                この操作は取り消せません。
              </p>
              <div className="flex gap-2 justify-end">
                <Button
                  variant="outline"
                  onClick={handleCancelDelete}
                >
                  キャンセル
                </Button>
                <Button
                  variant="destructive"
                  onClick={handleConfirmDelete}
                >
                  削除する
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}