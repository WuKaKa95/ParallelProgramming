with Ada.Text_IO;                use Ada.Text_IO;
with Ada.Numerics.Float_Random;  use Ada.Numerics.Float_Random;
with Random_Seeds;               use Random_Seeds;
with Ada.Real_Time;              use Ada.Real_Time;
with Ada.Characters.Handling;    use Ada.Characters.Handling;

procedure Travelers is

   -- Travelers configuration
   Nr_Of_Travelers      : constant Integer := 15;
   Nr_Of_Wild_Travelers : constant Integer := 10;
   Min_Steps            : constant Integer := 10;
   Max_Steps            : constant Integer := 100;
   Min_Delay            : constant Duration := 0.01;
   Max_Delay            : constant Duration := 0.05;
   Enter_Timeout        : constant Duration := 6 * Max_Delay;  -- For deadlock detection
   WildLifespan         : constant Duration := 0.5;

   -- Toroidal board dimensions
   Board_Width  : constant Integer := 15;
   Board_Height : constant Integer := 15;

   -- Global timing
   Start_Time : constant Time := Clock;
   Seeds      : constant Seed_Array_Type(1 .. Nr_Of_Travelers + Nr_Of_Wild_Travelers) := 
                Make_Seeds(Nr_Of_Travelers + Nr_Of_Wild_Travelers);

   -- Position representation
   type Position_Type is record
      X : Integer range 0 .. Board_Width;
      Y : Integer range 0 .. Board_Height;
   end record;

   -- Elementary movement operations
   procedure Move_Down(Position : in out Position_Type) is
   begin
      Position.Y := (Position.Y + 1) mod Board_Height;
   end Move_Down;

   procedure Move_Up(Position : in out Position_Type) is
   begin
      Position.Y := (Position.Y + Board_Height - 1) mod Board_Height;
   end Move_Up;

   procedure Move_Right(Position : in out Position_Type) is
   begin
      Position.X := (Position.X + 1) mod Board_Width;
   end Move_Right;

   procedure Move_Left(Position : in out Position_Type) is
   begin
      Position.X := (Position.X + Board_Width - 1) mod Board_Width;
   end Move_Left;

   -- Movement direction dispatcher
   procedure Move_Direction(Position : in out Position_Type; Direction : Integer) is
   begin
      case Direction is
         when 0 => Move_Up(Position);
         when 1 => Move_Down(Position);
         when 2 => Move_Left(Position);
         when 3 => Move_Right(Position);
         when others => Put_Line(" Invalid direction: " & Integer'Image(Direction));
      end case;
   end Move_Direction;

   -- Trace recording system
   type Trace_Type is record
      Time_Stamp : Duration;
      Id         : Integer;
      Position   : Position_Type;
      Symbol     : Character;
   end record;

   type Trace_Array_Type is array (0 .. Max_Steps) of Trace_Type;

   type Traces_Sequence_Type is record
      Last        : Integer := -1;
      Trace_Array : Trace_Array_Type;
   end record;

   procedure Print_Trace(Trace : Trace_Type) is
      Symbol_Str : String := (1 => Trace.Symbol);
   begin
      Put_Line(
         Duration'Image(Trace.Time_Stamp) & " " &
         Integer'Image(Trace.Id) & " " &
         Integer'Image(Trace.Position.X) & " " &
         Integer'Image(Trace.Position.Y) & " " &
         Symbol_Str
      );
   end Print_Trace;

   procedure Print_Traces(Traces : Traces_Sequence_Type) is
   begin
      for I in 0 .. Traces.Last loop
         Print_Trace(Traces.Trace_Array(I));
      end loop;
   end Print_Traces;

   -- Printer task collects all traces
   task Printer is
      entry Report(Traces : Traces_Sequence_Type);
   end Printer;

   task body Printer is
   begin
      for I in 1 .. Nr_Of_Travelers + Nr_Of_Wild_Travelers loop
         accept Report(Traces : Traces_Sequence_Type) do
            Print_Traces(Traces);
         end Report;
      end loop;
   end Printer;

   -- Traveler type definitions
   type Traveler_Variant is (Legal, Wild, None);

   type Traveler_Type is record
      Id       : Integer;
      Symbol   : Character;
      Position : Position_Type;
   end record;

   -- Task type declarations
   task type Traveler_Task_Type is
      entry Init(Id : Integer; Seed : Integer; Symbol : Character);
      entry Start;
   end Traveler_Task_Type;

   task type Wild_Traveler_Task_Type is
      entry Init(Id : Integer; Seed : Integer; Symbol : Character);
      entry Start;
      entry Relocate(New_Position : Position_Type);
   end Wild_Traveler_Task_Type;

   type General_Traveler_Task_Type (Variant : Traveler_Variant) is record
      case Variant is
         when Legal =>
            Legal_Task : Traveler_Task_Type;
         when Wild =>
            Wild_Task : Wild_Traveler_Task_Type;
         when None =>
            null;
      end case;
   end record;

   -- Board cell protection mechanism
   protected type Node is
      entry Init(New_Position : Position_Type);
      entry Enter(Traveler : access General_Traveler_Task_Type; Success : out Boolean);
      entry Move(New_Position : Position_Type; Success : out Boolean);
      entry Leave;
   private
      Inited    : Boolean := False;
      Occupant  : access General_Traveler_Task_Type;
      Position  : Position_Type;
   end Node;

   -- Global board and task structures
   Board : array (0 .. Board_Width - 1, 0 .. Board_Height - 1) of Node;
   Travel_Tasks : array (0 .. Nr_Of_Travelers + Nr_Of_Wild_Travelers - 1) of 
                  access General_Traveler_Task_Type;
   Null_Task : constant access General_Traveler_Task_Type := 
               new General_Traveler_Task_Type(Variant => None);

   -- Node implementation
   protected body Node is
      entry Init(New_Position : Position_Type) when not Inited is
      begin
         Position := New_Position;
         Occupant := Null_Task;
         Inited := True;
      end Init;

      entry Enter(Traveler : access General_Traveler_Task_Type; Success : out Boolean) 
        when Inited and Occupant.Variant /= Legal is
      begin
         if Occupant.Variant = None then
            Occupant := Traveler;
            Success := True;
         elsif Occupant.Variant = Wild and Traveler.Variant = Legal then
            declare
               New_Pos : Position_Type;
               Moved   : Boolean := False;
            begin
               for Dir in 0 .. 3 loop
                  New_Pos := Position;
                  Move_Direction(New_Pos, Dir);
                  select
                     Board(New_Pos.X, New_Pos.Y).Enter(Occupant, Moved);
                     exit when Moved;
                  else
                     null;
                  end select;
               end loop;
               if Moved then
                  Occupant.Wild_Task.Relocate(New_Pos);
                  Occupant := Traveler;
                  Success := True;
               else
                  Success := False;
               end if;
            end;
         else
            Success := False;
         end if;
      end Enter;

      entry Move(New_Position : Position_Type; Success : out Boolean) 
        when Inited and Occupant.Variant = Legal is
      begin
         select
            Board(New_Position.X, New_Position.Y).Enter(Occupant, Success);
         else
            Success := False;
         end select;
         if Success then
            Occupant := Null_Task;
         end if;
      end Move;

      entry Leave when Inited is
      begin
         Occupant := Null_Task;
      end Leave;
   end Node;

   -- Legal traveler implementation
   task body Traveler_Task_Type is
      G            : Generator;
      Traveler     : Traveler_Type;
      Time_Stamp   : Duration;
      Nr_of_Steps  : Integer;
      Traces       : Traces_Sequence_Type;

      procedure Store_Trace is
      begin
         Traces.Last := Traces.Last + 1;
         Traces.Trace_Array(Traces.Last) := (
            Time_Stamp => Time_Stamp,
            Id         => Traveler.Id,
            Position   => Traveler.Position,
            Symbol     => Traveler.Symbol
         );
      end Store_Trace;

      procedure Make_Step is
         Direction   : constant Integer := Integer(Float'Floor(4.0 * Random(G)));
         New_Pos     : Position_Type := Traveler.Position;
         Success     : Boolean;
         Deadlock    : Boolean := False;
      begin
         Move_Direction(New_Pos, Direction);
         select
            Board(New_Pos.X, New_Pos.Y).Enter(Travel_Tasks(Traveler.Id), Success);
         or
            delay Enter_Timeout;
            Deadlock := True;
         end select;

         if Deadlock then
            Traveler.Symbol := To_Lower(Traveler.Symbol);
            Time_Stamp := To_Duration(Clock - Start_Time);
            Store_Trace;
            raise Program_Error;
         elsif Success then
            Board(Traveler.Position.X, Traveler.Position.Y).Leave;
            Traveler.Position := New_Pos;
            Time_Stamp := To_Duration(Clock - Start_Time);
            Store_Trace;
         end if;
      end Make_Step;

   begin
accept Init(Id : Integer; Seed : Integer; Symbol : Character) do
   Reset(G, Seed);
   Traveler.Id := Id;
   Traveler.Symbol := Symbol;
   Nr_of_Steps := Min_Steps + Integer(Float(Max_Steps - Min_Steps) * Random(G));

   -- Initial position
   loop
      Traveler.Position := (
         X => Integer(Float'Floor(Float(Board_Width) * Random(G))),
         Y => Integer(Float'Floor(Float(Board_Height) * Random(G)))
      );
      declare
         Success : Boolean;
      begin
         select
            Board(Traveler.Position.X, Traveler.Position.Y)
              .Enter(Travel_Tasks(Traveler.Id), Success);
            exit when Success;
         or
            delay Enter_Timeout;
         end select;
      end;
   end loop;
   Time_Stamp := To_Duration(Clock - Start_Time);
   Store_Trace;
end Init;

      accept Start;

      begin
         for Step in 0 .. Nr_of_Steps loop
            delay Min_Delay + (Max_Delay - Min_Delay) * Duration(Random(G));
            Make_Step;
         end loop;
      exception
         when Program_Error => null;  -- Deadlock occurred
      end;
      Printer.Report(Traces);
   end Traveler_Task_Type;

   -- Wild traveler
   task body Wild_Traveler_Task_Type is
      G             : Generator;
      Traveler      : Traveler_Type;
      Time_Stamp    : Duration;
      Traces        : Traces_Sequence_Type;
      Time_Appear   : Duration;
      Time_Disappear: Duration;

      procedure Store_Trace is
      begin
         Traces.Last := Traces.Last + 1;
         Traces.Trace_Array(Traces.Last) := (
            Time_Stamp => Time_Stamp,
            Id         => Traveler.Id,
            Position   => Traveler.Position,
            Symbol     => Traveler.Symbol
         );
      end Store_Trace;

   begin
      accept Init(Id : Integer; Seed : Integer; Symbol : Character) do
         Reset(G, Seed);
         Traveler.Id := Id;
         Traveler.Symbol := Symbol;
         Time_Appear := Max_Delay * Min_Steps * Duration(Random(G));
         Time_Disappear := Time_Appear + WildLifespan;
      end Init;

      accept Start;

      delay Time_Appear;

      -- Initial position
      declare
         Success : Boolean := False;
      begin
         while not Success loop
            Traveler.Position := (
               X => Integer(Float'Floor(Float(Board_Width) * Random(G))),
               Y => Integer(Float'Floor(Float(Board_Height) * Random(G)))
            );
            select
               Board(Traveler.Position.X, Traveler.Position.Y)
                 .Enter(Travel_Tasks(Traveler.Id), Success);
            else
               null;
            end select;
         end loop;
         Time_Stamp := To_Duration(Clock - Start_Time);
         Store_Trace;
      end;

      loop
         select
            accept Relocate(New_Position : Position_Type) do
               Traveler.Position := New_Position;
            end Relocate;
            Time_Stamp := To_Duration(Clock - Start_Time);
            Store_Trace;
         or
            delay until Start_Time + To_Time_Span(Time_Disappear);
            exit;
         end select;
      end loop;

      Board(Traveler.Position.X, Traveler.Position.Y).Leave;
      Traveler.Position := (X => Board_Width, Y => Board_Height);
      Time_Stamp := To_Duration(Clock - Start_Time);
      Store_Trace;
      Printer.Report(Traces);
   end Wild_Traveler_Task_Type;

-- Main program initialization
begin
   Put_Line(
      "-1 " &
      Integer'Image(Nr_Of_Travelers + Nr_Of_Wild_Travelers) & " " &
      Integer'Image(Board_Width) & " " &
      Integer'Image(Board_Height)
   );

   -- Initialize board cells
   for X in 0 .. Board_Width - 1 loop
      for Y in 0 .. Board_Height - 1 loop
         Board(X, Y).Init((X => X, Y => Y));
      end loop;
   end loop;

   -- Create and initialize traveler tasks
   declare
      Symbol : Character := 'A';
      Id     : Integer := 0;
   begin
      for I in 1 .. Nr_Of_Travelers loop
         Travel_Tasks(Id) := new General_Traveler_Task_Type(Variant => Legal);
         Travel_Tasks(Id).Legal_Task.Init(Id, Seeds(Id + 1), Symbol);
         Symbol := Character'Succ(Symbol);
         Id := Id + 1;
      end loop;

      Symbol := '0';
      for I in 1 .. Nr_Of_Wild_Travelers loop
         Travel_Tasks(Id) := new General_Traveler_Task_Type(Variant => Wild);
         Travel_Tasks(Id).Wild_Task.Init(Id, Seeds(Id + 1), Symbol);
         Symbol := Character'Succ(Symbol);
         Id := Id + 1;
      end loop;
   end;

   -- Start all traveler tasks
   for T of Travel_Tasks loop
      case T.Variant is
         when Legal => T.Legal_Task.Start;
         when Wild  => T.Wild_Task.Start;
         when None  => null;
      end case;
   end loop;
end Travelers;
