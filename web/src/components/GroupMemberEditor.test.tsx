import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { GroupMemberEditor } from './GroupMemberEditor';
import type { Device } from '@/hooks/types';

// Mock languageHelper to always return 'en' for consistent test behavior
vi.mock('@/libs/languageHelper', () => ({
  isJapanese: vi.fn(() => false),
  getCurrentLocale: vi.fn(() => 'en')
}));

describe('GroupMemberEditor', () => {
  const mockDevices: Record<string, Device> = {
    'device1': {
      ip: '192.168.1.1',
      eoj: '0x029101',
      name: '029101[Single Function Lighting]',
      id: undefined,
      properties: {
        '81': { EDT: 'CA==', string: 'living' }, // Installation location: living (Base64 for [8])
      },
      lastSeen: '2024-01-01T00:00:00Z',
    },
    'device2': {
      ip: '192.168.1.2',
      eoj: '0x013001',
      name: '013001[Air Conditioner]',
      id: undefined,
      properties: {
        '81': { EDT: 'GA==', string: 'kitchen' }, // Installation location: kitchen (Base64 for [24])
      },
      lastSeen: '2024-01-01T00:00:00Z',
    },
    'device3': {
      ip: '192.168.1.3',
      eoj: '0x029101',
      name: '029101[Single Function Lighting]',
      id: undefined,
      properties: {
        '81': { EDT: 'QA==', string: 'room' }, // Installation location: bedroom/room (Base64 for [64])
      },
      lastSeen: '2024-01-01T00:00:00Z',
    },
  };

  const mockPropertyDescriptions = {
    '0291': {
      classCode: '0291',
      properties: {
        '81': {
          description: 'Installation Location',
          aliases: {
            'unspecified': 'AA==',  // Base64 for [0]
            'living': 'CA==',       // Base64 for [8]
            'kitchen': 'GA==',      // Base64 for [24]
            'bedroom': 'QA==',      // Base64 for [64] (room)
            'undetermined': '/w==', // Base64 for [255]
          },
          aliasTranslations: {
            'unspecified': '未指定',
            'living': 'リビング',
            'kitchen': 'キッチン',
            'bedroom': '寝室',
            'room': '部屋',
            'undetermined': '未定'
          }
        }
      }
    }
  };

  const defaultProps = {
    groupName: '@testgroup',
    groupMembers: ['device1'],
    allDevices: mockDevices,
    aliases: {
      'リビングライト': 'device1',
      'キッチン照明': 'device2',
    },
    onAddToGroup: vi.fn(),
    onRemoveFromGroup: vi.fn(),
    propertyDescriptions: mockPropertyDescriptions,
    getDeviceClassCode: vi.fn().mockReturnValue('0291'),
    isLoading: false,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render group members section and available devices section', () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    expect(screen.getByText('@testgroup のメンバー')).toBeInTheDocument();
    expect(screen.getByText('利用可能なデバイス')).toBeInTheDocument();
  });

  it('should display group members in the top section', () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const membersSection = screen.getByTestId('group-members-section');
    expect(membersSection).toHaveTextContent('029101[Single Function Lighting]');
    expect(membersSection).toHaveTextContent('192.168.1.1 0x029101');
    // Installation location should be displayed
    expect(membersSection).toHaveTextContent('設置場所: living');
  });

  it('should display installation location for each device', () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    // Check if location is displayed for member device
    const membersSection = screen.getByTestId('group-members-section');
    expect(membersSection).toHaveTextContent('設置場所: living');
    
    // Check if location is displayed for available devices  
    const availableSection = screen.getByTestId('available-devices-section');
    expect(availableSection).toHaveTextContent('設置場所: kitchen');
    expect(availableSection).toHaveTextContent('設置場所: room');
  });

  it('should display non-member devices in the bottom section', () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const availableSection = screen.getByTestId('available-devices-section');
    expect(availableSection).toHaveTextContent('013001[Air Conditioner]');
    expect(availableSection).toHaveTextContent('029101[Single Function Lighting]');
    expect(availableSection).toHaveTextContent('設置場所: kitchen');
    expect(availableSection).toHaveTextContent('設置場所: room');
  });


  it('should make device cards draggable', () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const deviceCards = screen.getAllByTestId(/^device-card-/);
    deviceCards.forEach(card => {
      expect(card).toHaveAttribute('draggable', 'true');
    });
  });

  it('should handle drag start event', () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const deviceCard = screen.getByTestId('device-card-device1');
    const dataTransfer = {
      setData: vi.fn(),
      effectAllowed: '',
    };
    
    fireEvent.dragStart(deviceCard, { dataTransfer });
    
    expect(dataTransfer.setData).toHaveBeenCalledWith('text/plain', 'device1');
    expect(dataTransfer.effectAllowed).toBe('move');
  });

  it('should handle drop event to add device to group', async () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const membersSection = screen.getByTestId('group-members-section');
    const dataTransfer = {
      getData: vi.fn().mockReturnValue('device2'),
    };
    
    fireEvent.dragOver(membersSection, { preventDefault: vi.fn() });
    fireEvent.drop(membersSection, { 
      preventDefault: vi.fn(),
      dataTransfer,
    });
    
    expect(defaultProps.onAddToGroup).toHaveBeenCalledWith('@testgroup', ['device2']);
  });

  it('should handle drop event to remove device from group', async () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const availableSection = screen.getByTestId('available-devices-section');
    const dataTransfer = {
      getData: vi.fn().mockReturnValue('device1'),
    };
    
    fireEvent.dragOver(availableSection, { preventDefault: vi.fn() });
    fireEvent.drop(availableSection, { 
      preventDefault: vi.fn(),
      dataTransfer,
    });
    
    expect(defaultProps.onRemoveFromGroup).toHaveBeenCalledWith('@testgroup', ['device1']);
  });

  it('should show visual feedback during drag over', () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const membersSection = screen.getByTestId('group-members-section');
    
    fireEvent.dragEnter(membersSection);
    expect(membersSection).toHaveClass('drag-over');
    
    fireEvent.dragLeave(membersSection);
    expect(membersSection).not.toHaveClass('drag-over');
  });

  it('should not allow dropping device on itself', () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const membersSection = screen.getByTestId('group-members-section');
    const dataTransfer = {
      getData: vi.fn().mockReturnValue('device1'), // Already a member
    };
    
    fireEvent.drop(membersSection, { 
      preventDefault: vi.fn(),
      dataTransfer,
    });
    
    // Should not call onAddToGroup for a device already in the group
    expect(defaultProps.onAddToGroup).not.toHaveBeenCalled();
  });

  it('should show empty state when no devices available', () => {
    render(<GroupMemberEditor {...defaultProps} allDevices={{}} />);
    
    expect(screen.getByText('利用可能なデバイスがありません')).toBeInTheDocument();
  });

  it('should show empty state for group with no members', () => {
    render(<GroupMemberEditor {...defaultProps} groupMembers={[]} />);
    
    const membersSection = screen.getByTestId('group-members-section');
    expect(membersSection).toHaveTextContent('デバイスをここにドラッグしてグループに追加');
  });

  it('should display minus button for member devices', () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const removeButton = screen.getByTestId('remove-device-device1');
    expect(removeButton).toBeInTheDocument();
    expect(removeButton).toHaveAttribute('title', 'グループから削除');
  });

  it('should display plus button for available devices', () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const addButton2 = screen.getByTestId('add-device-device2');
    const addButton3 = screen.getByTestId('add-device-device3');
    expect(addButton2).toBeInTheDocument();
    expect(addButton3).toBeInTheDocument();
    expect(addButton2).toHaveAttribute('title', 'グループに追加');
    expect(addButton3).toHaveAttribute('title', 'グループに追加');
  });

  it('should call onRemoveFromGroup when minus button is clicked', async () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const removeButton = screen.getByTestId('remove-device-device1');
    fireEvent.click(removeButton);
    
    expect(defaultProps.onRemoveFromGroup).toHaveBeenCalledWith('@testgroup', ['device1']);
  });

  it('should call onAddToGroup when plus button is clicked', async () => {
    render(<GroupMemberEditor {...defaultProps} />);
    
    const addButton = screen.getByTestId('add-device-device2');
    fireEvent.click(addButton);
    
    expect(defaultProps.onAddToGroup).toHaveBeenCalledWith('@testgroup', ['device2']);
  });

  it('should disable buttons when isLoading is true', () => {
    render(<GroupMemberEditor {...defaultProps} isLoading={true} />);
    
    const removeButton = screen.getByTestId('remove-device-device1');
    const addButton = screen.getByTestId('add-device-device2');
    
    expect(removeButton).toBeDisabled();
    expect(addButton).toBeDisabled();
  });

  it('should handle unspecified location properly', () => {
    const deviceWithUnspecifiedLocation = {
      ...mockDevices.device1,
      id: undefined,
      properties: {
        '81': { EDT: 'AA==', string: 'unspecified' } // unspecified: Base64 for [0]
      }
    };

    const propsWithUnspecified = {
      ...defaultProps,
      allDevices: {
        ...defaultProps.allDevices,
        device1: deviceWithUnspecifiedLocation
      }
    };

    render(<GroupMemberEditor {...propsWithUnspecified} />);
    
    const membersSection = screen.getByTestId('group-members-section');
    expect(membersSection).toHaveTextContent('設置場所: unspecified');
  });
});