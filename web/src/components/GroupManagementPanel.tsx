import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Edit2, Users, Trash2 } from 'lucide-react';

interface GroupManagementPanelProps {
  groupName: string;
  onRename: () => void;
  onDelete: () => void;
  onEditMembers: () => void;
  isEditingMembers?: boolean;
  onDoneEditingMembers?: () => void;
}

export function GroupManagementPanel({
  groupName,
  onRename,
  onDelete,
  onEditMembers,
  isEditingMembers = false,
  onDoneEditingMembers,
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
      <Card>
        <CardContent className="flex flex-wrap gap-2">
          {isEditingMembers ? (
            <Button
              variant="outline"
              size="sm"
              onClick={onDoneEditingMembers}
              title="メンバー編集を終了"
            >
              <Users className="h-4 w-4 sm:mr-2" />
              <span className="hidden sm:inline">編集を終了</span>
            </Button>
          ) : (
            <>
              <Button
                variant="outline"
                size="sm"
                onClick={onRename}
                title="グループ名を変更"
              >
                <Edit2 className="h-4 w-4 sm:mr-2" />
                <span className="hidden sm:inline">グループ名を変更</span>
              </Button>
              
              <Button
                variant="outline"
                size="sm"
                onClick={onEditMembers}
                title="メンバーを編集"
              >
                <Users className="h-4 w-4 sm:mr-2" />
                <span className="hidden sm:inline">メンバーを編集</span>
              </Button>
              
              <Button
                variant="outline"
                size="sm"
                className="destructive"
                onClick={handleDeleteClick}
                title="グループを削除"
              >
                <Trash2 className="h-4 w-4 sm:mr-2" />
                <span className="hidden sm:inline">グループを削除</span>
              </Button>
            </>
          )}
        </CardContent>
      </Card>

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