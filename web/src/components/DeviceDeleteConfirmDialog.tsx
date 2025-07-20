import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { isJapanese } from '@/libs/languageHelper';
import type { Device } from '@/hooks/types';

type DialogMessages = {
  title: string;
  description: string;
  warning: string;
  cancel: string;
  delete: string;
  deleting: string;
};

interface DeviceDeleteConfirmDialogProps {
  device: Device;
  aliasName?: string;
  onDeleteDevice: (target: string) => Promise<void>;
  isDeletingDevice: boolean;
  isConnected: boolean;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function DeviceDeleteConfirmDialog({
  device,
  aliasName,
  onDeleteDevice,
  isDeletingDevice,
  isConnected,
  isOpen,
  onOpenChange,
}: DeviceDeleteConfirmDialogProps) {
  const deviceDisplayName = aliasName || device.name;
  const deviceTarget = `${device.ip} ${device.eoj}`;

  const messages: Record<'en' | 'ja', DialogMessages> = {
    en: {
      title: 'Delete Offline Device',
      description: `Are you sure you want to delete "${deviceDisplayName}"?`,
      warning: 'This action cannot be undone. The device will be permanently removed from the device list.',
      cancel: 'Cancel',
      delete: 'Delete Device',
      deleting: 'Deleting...',
    },
    ja: {
      title: 'オフラインデバイスを削除',
      description: `「${deviceDisplayName}」を削除してもよろしいですか？`,
      warning: 'この操作は取り消すことができません。デバイスはデバイスリストから完全に削除されます。',
      cancel: 'キャンセル',
      delete: 'デバイスを削除',
      deleting: '削除中...',
    },
  };

  const texts = isJapanese() ? messages.ja : messages.en;

  return (
    <AlertDialog open={isOpen} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{texts.title}</AlertDialogTitle>
          <AlertDialogDescription>
            {texts.description}
            <br />
            <span className="text-xs text-muted-foreground mt-1 block">
              {device.ip} - {device.eoj}
            </span>
            <br />
            {texts.warning}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isDeletingDevice}>
            {texts.cancel}
          </AlertDialogCancel>
          <AlertDialogAction
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            onClick={() => onDeleteDevice(deviceTarget)}
            disabled={isDeletingDevice || !isConnected}
          >
            {isDeletingDevice ? texts.deleting : texts.delete}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}