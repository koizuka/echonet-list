import { DummyDeviceList } from '@/components/DummyDeviceList';

function App() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="container mx-auto p-4">
        <h1 className="text-3xl font-bold mb-6">ECHONET List</h1>
        <DummyDeviceList />
      </div>
    </div>
  );
}

export default App;