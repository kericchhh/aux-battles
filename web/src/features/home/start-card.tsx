import { useState } from 'react'

import { Card } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { HostForm } from './host-form'
import { JoinForm } from './join-form'

const modes = ['host', 'join'] as const
type Mode = (typeof modes)[number]

export function StartCard({ className }: { className?: string }) {
  const [mode, setMode] = useState<Mode>('host')

  return (
    <Card title="Start" className={className}>
      <Tabs value={mode} onValueChange={(value) => setMode(value as Mode)}>
        <TabsList aria-label="Start a battle" count={modes.length} selected={modes.indexOf(mode)}>
          <TabsTrigger value="host">Host</TabsTrigger>
          <TabsTrigger value="join">Join</TabsTrigger>
        </TabsList>
        <TabsContent value="host" className="tab-panel">
          <HostForm />
        </TabsContent>
        <TabsContent value="join" className="tab-panel">
          <JoinForm />
        </TabsContent>
      </Tabs>
    </Card>
  )
}
